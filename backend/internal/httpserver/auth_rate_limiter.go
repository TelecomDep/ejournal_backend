package httpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Authentication endpoints are commonly used by a whole class or office
// behind one NAT address.  The aggregate limits therefore have to be high
// enough for normal shared-IP traffic, while the per-account/per-credential
// limits still make password and invite-code guessing expensive.
const (
	authRateLimitWindow = 5 * time.Minute

	loginAggregateLimit         = 600
	loginAccountFailureLimit    = 10
	registrationAggregateLimit  = 300
	registrationIPFailureLimit  = 20
	registrationLoginFailLimit  = 10
	registrationInviteFailLimit = 10

	authRateLimitMaxKeys = 8192
)

type authRateLimitMode uint8

const (
	authLoginMode authRateLimitMode = iota + 1
	authRegistrationMode
)

// authRateLimiterConfig is intentionally kept private.  The constructors
// below provide production defaults, while tests in this package can use a
// short window and a controllable clock without relying on sleeps.
type authRateLimiterConfig struct {
	mode                authRateLimitMode
	window              time.Duration
	aggregateLimit      int
	accountFailureLimit int
	ipFailureLimit      int
	loginFailureLimit   int
	inviteFailureLimit  int
	maxKeys             int
	now                 func() time.Time
}

// authRateLimiter applies two kinds of protection:
//
//   - every request consumes a generous per-IP aggregate slot, so rotating
//     account names cannot create an unbounded stream of guesses;
//   - failed credentials consume a separate key after the downstream handler
//     has classified the result.  A successful login clears its account
//     failure window; a requires_2fa challenge neither adds nor clears a
//     failure window because only the password has been validated.
//
// Keeping failure accounting after the handler is important: counting every
// request would make a busy shared NAT run out of login capacity even when all
// credentials are valid.
type authRateLimiter struct {
	config authRateLimiterConfig
	store  *authRateWindowStore
}

type authRateWindow struct {
	hits         []time.Time
	reservations int
	lastSeen     time.Time
}

type authRateReservation struct {
	key    string
	limit  int
	active bool
}

// authRateWindowStore is a bounded, in-memory sliding-window counter.  A
// bounded store avoids allowing attacker-controlled login names or invite
// codes to grow process memory without limit.  The aggregate per-IP ceiling
// remains the primary protection when old keys are evicted.
type authRateWindowStore struct {
	mu      sync.Mutex
	windows map[string]*authRateWindow
	window  time.Duration
	maxKeys int
	now     func() time.Time
}

func newAuthRateWindowStore(window time.Duration, maxKeys int, now func() time.Time) *authRateWindowStore {
	if window <= 0 {
		window = authRateLimitWindow
	}
	if maxKeys <= 0 {
		maxKeys = authRateLimitMaxKeys
	}
	if now == nil {
		now = time.Now
	}
	return &authRateWindowStore{
		windows: make(map[string]*authRateWindow),
		window:  window,
		maxKeys: maxKeys,
		now:     now,
	}
}

func (s *authRateWindowStore) currentTime() time.Time {
	if s.now == nil {
		return time.Now()
	}
	return s.now()
}

func (s *authRateWindowStore) pruneLocked(key string, now time.Time) *authRateWindow {
	window, ok := s.windows[key]
	if !ok {
		return nil
	}

	cutoff := now.Add(-s.window)
	firstLive := 0
	for firstLive < len(window.hits) && window.hits[firstLive].Before(cutoff) {
		firstLive++
	}
	if firstLive > 0 {
		copy(window.hits, window.hits[firstLive:])
		window.hits = window.hits[:len(window.hits)-firstLive]
	}
	window.lastSeen = now
	if len(window.hits) == 0 && window.reservations == 0 {
		delete(s.windows, key)
		return nil
	}
	return window
}

func (s *authRateWindowStore) evictOneLocked() bool {
	if len(s.windows) < s.maxKeys {
		return true
	}

	var oldestKey string
	var oldest time.Time
	for key, window := range s.windows {
		if window.reservations > 0 {
			continue
		}
		if oldestKey == "" || window.lastSeen.Before(oldest) {
			oldestKey = key
			oldest = window.lastSeen
		}
	}
	if oldestKey != "" {
		delete(s.windows, oldestKey)
		return true
	}
	return false
}

// reserve atomically holds one failure slot until the downstream handler
// classifies the response.  The reservation is committed only for a bad
// credential; successful and operational responses release it.
func (s *authRateWindowStore) reserve(key string, limit int, now time.Time, includePending bool) (*authRateReservation, bool) {
	if key == "" {
		return nil, false
	}
	if limit <= 0 {
		// A zero optional failure limit disables that key.  Keep the
		// reservation list shape stable so registration identity keys can
		// still be cleared independently of the per-IP key.
		return nil, true
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	window := s.pruneLocked(key, now)
	if window != nil {
		used := len(window.hits)
		if includePending {
			used += window.reservations
		}
		if used >= limit {
			return nil, false
		}
	}
	if window == nil {
		if !s.evictOneLocked() {
			return nil, false
		}
		window = &authRateWindow{}
		s.windows[key] = window
	}
	window.lastSeen = now
	window.reservations++
	return &authRateReservation{key: key, limit: limit, active: true}, true
}

// try consumes one slot and reports whether it was available.  Aggregate
// request windows use this method because every request, including successful
// ones, must contribute to the per-IP ceiling.
func (s *authRateWindowStore) try(key string, limit int, now time.Time) bool {
	if key == "" || limit <= 0 {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	window := s.pruneLocked(key, now)
	if window != nil && len(window.hits) >= limit {
		return false
	}
	if window == nil {
		if !s.evictOneLocked() {
			return false
		}
		window = &authRateWindow{}
		s.windows[key] = window
	}
	window.lastSeen = now
	window.hits = append(window.hits, now)
	return true
}

func (s *authRateWindowStore) finalize(reservation *authRateReservation, retain bool, now time.Time) {
	if reservation == nil || !reservation.active {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	reservation.active = false

	window := s.pruneLocked(reservation.key, now)
	if window == nil {
		// This should only be possible after an unexpected eviction.  The
		// reservation is still considered released/consumed for this
		// request; failing closed is safer than recreating unbounded state.
		return
	}
	if window.reservations > 0 {
		window.reservations--
	}
	window.lastSeen = now
	if retain && len(window.hits) < reservation.limit {
		window.hits = append(window.hits, now)
	}
	if len(window.hits) == 0 && window.reservations == 0 {
		delete(s.windows, reservation.key)
	}
}

func (s *authRateWindowStore) clear(key string) {
	if key == "" {
		return
	}
	s.mu.Lock()
	if window := s.windows[key]; window != nil && window.reservations > 0 {
		// Preserve in-flight reservations while forgiving completed failures.
		window.hits = nil
	} else {
		delete(s.windows, key)
	}
	s.mu.Unlock()
}

func newAuthLoginLimiter() fiber.Handler {
	limiter := newAuthRateLimiter(authRateLimiterConfig{
		mode:                authLoginMode,
		window:              authRateLimitWindow,
		aggregateLimit:      loginAggregateLimit,
		accountFailureLimit: loginAccountFailureLimit,
		maxKeys:             authRateLimitMaxKeys,
	})
	return limiter.handle
}

func newAuthRegistrationLimiter() fiber.Handler {
	limiter := newAuthRateLimiter(authRateLimiterConfig{
		mode:               authRegistrationMode,
		window:             authRateLimitWindow,
		aggregateLimit:     registrationAggregateLimit,
		ipFailureLimit:     registrationIPFailureLimit,
		loginFailureLimit:  registrationLoginFailLimit,
		inviteFailureLimit: registrationInviteFailLimit,
		maxKeys:            authRateLimitMaxKeys,
	})
	return limiter.handle
}

func newAuthRateLimiter(config authRateLimiterConfig) *authRateLimiter {
	if config.window <= 0 {
		config.window = authRateLimitWindow
	}
	if config.maxKeys <= 0 {
		config.maxKeys = authRateLimitMaxKeys
	}
	if config.now == nil {
		config.now = time.Now
	}
	return &authRateLimiter{
		config: config,
		store:  newAuthRateWindowStore(config.window, config.maxKeys, config.now),
	}
}

func (l *authRateLimiter) handle(c *fiber.Ctx) error {
	if l == nil || l.store == nil {
		return c.Next()
	}

	now := l.store.currentTime()
	ip := authRequestIP(c)
	if !l.store.try(l.aggregateKey(ip), l.config.aggregateLimit, now) {
		return tooManyAuthAttempts(c)
	}
	if len(c.Body()) > 64*1024 {
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(map[string]any{
			"ok": false, "error": "authentication request is too large",
		})
	}

	attempt := parseAuthAttempt(c)
	failureKeys, failureLimits := l.failureKeys(ip, attempt)
	reservations := make([]*authRateReservation, 0, len(failureKeys))
	for i, key := range failureKeys {
		// A shared IP failure budget counts completed failures, not pending
		// registrations from an entire classroom. Credential-specific budgets
		// still reserve slots, and the aggregate IP cap bounds concurrent load.
		includePending := !(l.config.mode == authRegistrationMode && i == 0)
		reservation, ok := l.store.reserve(key, failureLimits[i], now, includePending)
		if !ok {
			for _, held := range reservations {
				l.store.finalize(held, false, now)
			}
			return tooManyAuthAttempts(c)
		}
		reservations = append(reservations, reservation)
	}

	// Release pending slots even if a downstream panic is handled by Fiber's
	// outer recovery middleware. Already finalized reservations are no-ops.
	defer func() {
		for _, reservation := range reservations {
			l.store.finalize(reservation, false, l.store.currentTime())
		}
	}()
	err := c.Next()
	if err != nil {
		for _, reservation := range reservations {
			l.store.finalize(reservation, false, now)
		}
		return err
	}

	ok, responseError, parsed := authResponse(c)
	if !parsed {
		for _, reservation := range reservations {
			l.store.finalize(reservation, false, now)
		}
		return nil
	}

	switch l.config.mode {
	case authLoginMode:
		accountKey := loginFailureKey(attempt.Login)
		if accountKey == "" {
			for _, reservation := range reservations {
				l.store.finalize(reservation, false, now)
			}
			return nil
		}
		switch {
		case ok:
			// A fully successful login is not a failed credential and should
			// forgive earlier mistakes.
			for _, reservation := range reservations {
				l.store.finalize(reservation, false, now)
			}
			l.store.clear(accountKey)
		case strings.EqualFold(responseError, "requires_2fa"):
			// Do not clear earlier bad-2FA failures here.  This response only
			// says that the password was correct and a second factor is still
			// required; otherwise an attacker could alternate empty and bad
			// codes to reset the account's failure budget.
			for _, reservation := range reservations {
				l.store.finalize(reservation, false, now)
			}
		case isLoginCredentialFailure(responseError):
			for _, reservation := range reservations {
				l.store.finalize(reservation, true, now)
			}
		default:
			for _, reservation := range reservations {
				l.store.finalize(reservation, false, now)
			}
		}
	case authRegistrationMode:
		keys, _ := l.failureKeys(ip, attempt)
		if ok {
			for _, reservation := range reservations {
				l.store.finalize(reservation, false, now)
			}
			// A successful registration should not forgive failed invite
			// guesses made by other users behind the same NAT.  Only clear
			// identity-specific windows belonging to this request.
			for _, key := range keys[1:] {
				l.store.clear(key)
			}
		} else if isRegistrationFailure(responseError) {
			for _, reservation := range reservations {
				l.store.finalize(reservation, true, now)
			}
		} else {
			for _, reservation := range reservations {
				l.store.finalize(reservation, false, now)
			}
		}
	default:
		for _, reservation := range reservations {
			l.store.finalize(reservation, false, now)
		}
	}

	return nil
}

func (l *authRateLimiter) aggregateKey(ip string) string {
	return hashedAuthKey("aggregate|", ip)
}

func (l *authRateLimiter) failureKeys(ip string, attempt authAttempt) ([]string, []int) {
	switch l.config.mode {
	case authLoginMode:
		if key := loginFailureKey(attempt.Login); key != "" {
			return []string{key}, []int{l.config.accountFailureLimit}
		}
	case authRegistrationMode:
		keys := []string{hashedAuthKey("registration-ip|", ip)}
		limits := []int{l.config.ipFailureLimit}
		if login := normalizeAuthIdentity(attempt.Login); login != "" {
			keys = append(keys, hashedAuthKey("registration-login|", login))
			limits = append(limits, l.config.loginFailureLimit)
		}
		if invite := normalizeInviteIdentity(attempt.InviteCode); invite != "" {
			keys = append(keys, hashedAuthKey("registration-invite|", invite))
			limits = append(limits, l.config.inviteFailureLimit)
		}
		return keys, limits
	}
	return nil, nil
}

type authAttempt struct {
	Login      string
	InviteCode string
}

// parseAuthAttempt reads the request body without consuming it.  Fiber's
// BodyParser supports JSON, form-encoded, and multipart requests and the
// downstream handler can parse the same payload afterwards.
func parseAuthAttempt(c *fiber.Ctx) authAttempt {
	body := c.Body()
	if len(body) == 0 || len(body) > 64*1024 {
		return authAttempt{}
	}

	var payload struct {
		Login      string `json:"login"`
		InviteCode string `json:"invite_code"`
	}
	if c.BodyParser(&payload) != nil {
		return authAttempt{}
	}

	return authAttempt{
		Login:      normalizeAuthIdentity(payload.Login),
		InviteCode: normalizeInviteIdentity(payload.InviteCode),
	}
}

func authRequestIP(c *fiber.Ctx) string {
	ip := strings.TrimSpace(c.IP())
	if ip == "" {
		return "unknown"
	}
	return ip
}

func normalizeAuthIdentity(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeInviteIdentity(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func loginFailureKey(login string) string {
	login = normalizeAuthIdentity(login)
	if login == "" {
		return ""
	}
	return hashedAuthKey("login-account|", login)
}

// hashedAuthKey keeps attacker-controlled account names, invite codes, and
// proxy-derived addresses out of the map itself.  Besides bounding each key
// to a fixed size, this avoids retaining invitation secrets for the lifetime
// of a failure window.
func hashedAuthKey(prefix, value string) string {
	digest := sha256.Sum256([]byte(value))
	return prefix + hex.EncodeToString(digest[:])
}

func authResponse(c *fiber.Ctx) (ok bool, responseError string, parsed bool) {
	var response struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if len(c.Response().Body()) == 0 || json.Unmarshal(c.Response().Body(), &response) != nil {
		return false, "", false
	}
	return response.OK, strings.TrimSpace(response.Error), true
}

func isLoginCredentialFailure(responseError string) bool {
	switch strings.ToLower(strings.TrimSpace(responseError)) {
	case "invalid credentials", "invalid 2fa code":
		return true
	default:
		return false
	}
}

func isRegistrationFailure(responseError string) bool {
	if strings.TrimSpace(responseError) == "" {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(responseError))
	// Database, hashing, and token failures are operational errors rather than
	// guesses.  They must not lock out a shared IP during an outage.
	if strings.HasPrefix(message, "failed ") || strings.HasPrefix(message, "eror_") {
		return false
	}
	return true
}

func tooManyAuthAttempts(c *fiber.Ctx) error {
	c.Set(fiber.HeaderRetryAfter, "300")
	return c.Status(fiber.StatusTooManyRequests).JSON(map[string]any{
		"ok":    false,
		"error": "too many authentication attempts",
	})
}
