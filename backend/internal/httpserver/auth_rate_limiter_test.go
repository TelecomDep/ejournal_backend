package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func testAuthRateLimiter(t *testing.T, mode authRateLimitMode, aggregate, failure int) (fiber.Handler, func(time.Duration)) {
	t.Helper()
	var mu sync.Mutex
	now := time.Unix(100, 0)
	config := authRateLimiterConfig{
		mode:                mode,
		window:              time.Minute,
		aggregateLimit:      aggregate,
		accountFailureLimit: failure,
		ipFailureLimit:      failure,
		loginFailureLimit:   failure,
		inviteFailureLimit:  failure,
		maxKeys:             512,
		now: func() time.Time {
			mu.Lock()
			defer mu.Unlock()
			return now
		},
	}
	limiter := newAuthRateLimiter(config)
	advance := func(by time.Duration) {
		mu.Lock()
		now = now.Add(by)
		mu.Unlock()
	}
	return limiter.handle, advance
}

func sendAuthRequest(t *testing.T, app *fiber.App, path, login, invite string) *http.Response {
	t.Helper()
	body, err := json.Marshal(map[string]string{
		"login":       login,
		"password":    "not-a-real-password",
		"invite_code": invite,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	request := httptest.NewRequest("POST", path, strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	return response
}

func newAuthTestApp(t *testing.T, limiter fiber.Handler, path string, responseOK bool, responseError string) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Post(path, limiter, func(c *fiber.Ctx) error {
		return c.JSON(map[string]any{"ok": responseOK, "error": responseError})
	})
	return app
}

func TestAuthLoginRateLimiterAllowsClassSizedSharedIPSuccesses(t *testing.T) {
	limiter, _ := testAuthRateLimiter(t, authLoginMode, 120, 10)
	app := newAuthTestApp(t, limiter, "/login", true, "")

	for i := 0; i < 40; i++ {
		response := sendAuthRequest(t, app, "/login", fmt.Sprintf("student-%02d", i), "")
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("distinct valid user %d was blocked with status %d", i, response.StatusCode)
		}
	}
}

func TestAuthLoginRateLimiterBlocksRepeatedBadCredentialsPerAccount(t *testing.T) {
	limiter, _ := testAuthRateLimiter(t, authLoginMode, 100, 3)
	app := newAuthTestApp(t, limiter, "/login", false, "invalid credentials")

	for attempt := 0; attempt < 3; attempt++ {
		response := sendAuthRequest(t, app, "/login", "Student-1", "")
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("credential failure %d unexpectedly blocked with status %d", attempt+1, response.StatusCode)
		}
	}
	response := sendAuthRequest(t, app, "/login", "student-1", "")
	if response.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("fourth failure status = %d, want %d", response.StatusCode, fiber.StatusTooManyRequests)
	}
}

func TestAuthLoginRateLimiterReservesConcurrentFailureSlots(t *testing.T) {
	limiter, _ := testAuthRateLimiter(t, authLoginMode, 100, 2)
	app := newAuthTestApp(t, limiter, "/login", false, "invalid credentials")

	const concurrentAttempts = 8
	start := make(chan struct{})
	statuses := make(chan int, concurrentAttempts)
	var wg sync.WaitGroup
	for i := 0; i < concurrentAttempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			response := sendAuthRequest(t, app, "/login", "same-account", "")
			statuses <- response.StatusCode
		}()
	}
	close(start)
	wg.Wait()
	close(statuses)

	allowed := 0
	blocked := 0
	for status := range statuses {
		switch status {
		case fiber.StatusOK:
			allowed++
		case fiber.StatusTooManyRequests:
			blocked++
		default:
			t.Fatalf("unexpected concurrent response status %d", status)
		}
	}
	if allowed > 2 || blocked == 0 {
		t.Fatalf("concurrent attempts allowed=%d blocked=%d, want at most 2 allowed and at least 1 blocked", allowed, blocked)
	}
}

func TestAuthLoginRateLimiterSuccessAndTwoFAChallengeDoNotConsumeFailureWindow(t *testing.T) {
	limiter, _ := testAuthRateLimiter(t, authLoginMode, 100, 3)
	app := fiber.New()
	app.Post("/login", limiter, func(c *fiber.Ctx) error {
		var payload struct {
			Login string `json:"login"`
		}
		if err := c.BodyParser(&payload); err != nil {
			return err
		}
		if payload.Login == "challenge-user" {
			return c.JSON(map[string]any{"ok": false, "error": "requires_2fa"})
		}
		return c.JSON(map[string]any{"ok": true, "error": ""})
	})

	for i := 0; i < 20; i++ {
		response := sendAuthRequest(t, app, "/login", fmt.Sprintf("good-%02d", i), "")
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("successful login %d unexpectedly blocked with status %d", i, response.StatusCode)
		}
	}
	for i := 0; i < 3; i++ {
		response := sendAuthRequest(t, app, "/login", "challenge-user", "")
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("2FA challenge %d unexpectedly blocked with status %d", i+1, response.StatusCode)
		}
	}

	// A valid response clears earlier failures for the same normalized account.
	badApp := newAuthTestApp(t, limiter, "/login", false, "invalid credentials")
	for i := 0; i < 2; i++ {
		response := sendAuthRequest(t, badApp, "/login", "Mixed-Case", "")
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("pre-reset failure %d unexpectedly blocked with status %d", i+1, response.StatusCode)
		}
	}
	goodApp := newAuthTestApp(t, limiter, "/login", true, "")
	if response := sendAuthRequest(t, goodApp, "/login", "mixed-case", ""); response.StatusCode != fiber.StatusOK {
		t.Fatalf("successful reset was blocked with status %d", response.StatusCode)
	}
	for i := 0; i < 3; i++ {
		response := sendAuthRequest(t, badApp, "/login", "MIXED-CASE", "")
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("post-reset failure %d unexpectedly blocked with status %d", i+1, response.StatusCode)
		}
	}
}

func TestAuthLoginRateLimiterCountsBadTwoFAAsCredentialFailure(t *testing.T) {
	limiter, _ := testAuthRateLimiter(t, authLoginMode, 100, 2)
	app := newAuthTestApp(t, limiter, "/login", false, "invalid 2fa code")

	for i := 0; i < 2; i++ {
		response := sendAuthRequest(t, app, "/login", "two-fa-user", "")
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("bad 2FA attempt %d unexpectedly blocked with status %d", i+1, response.StatusCode)
		}
	}
	response := sendAuthRequest(t, app, "/login", "two-fa-user", "")
	if response.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("third bad 2FA status = %d, want %d", response.StatusCode, fiber.StatusTooManyRequests)
	}
}

func TestAuthLoginRateLimiterChallengeDoesNotClearBadTwoFAFailures(t *testing.T) {
	limiter, _ := testAuthRateLimiter(t, authLoginMode, 100, 2)
	call := 0
	app := fiber.New()
	app.Post("/login", limiter, func(c *fiber.Ctx) error {
		call++
		if call == 2 {
			return c.JSON(map[string]any{"ok": false, "error": "requires_2fa"})
		}
		return c.JSON(map[string]any{"ok": false, "error": "invalid 2fa code"})
	})

	for i := 0; i < 3; i++ {
		response := sendAuthRequest(t, app, "/login", "two-fa-user", "")
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("attempt %d unexpectedly blocked with status %d", i+1, response.StatusCode)
		}
	}
	response := sendAuthRequest(t, app, "/login", "two-fa-user", "")
	if response.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("fourth attempt after challenge status = %d, want %d", response.StatusCode, fiber.StatusTooManyRequests)
	}
}

func TestAuthLoginRateLimiterAggregateCapCannotBeBypassedByFreshIdentities(t *testing.T) {
	limiter, _ := testAuthRateLimiter(t, authLoginMode, 4, 100)
	app := newAuthTestApp(t, limiter, "/login", true, "")

	for i := 0; i < 4; i++ {
		response := sendAuthRequest(t, app, "/login", fmt.Sprintf("fresh-%d", i), "")
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("fresh identity %d unexpectedly blocked with status %d", i, response.StatusCode)
		}
	}
	response := sendAuthRequest(t, app, "/login", "fresh-beyond-cap", "")
	if response.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("fresh identity beyond aggregate cap status = %d, want %d", response.StatusCode, fiber.StatusTooManyRequests)
	}
}

func TestAuthRegistrationRateLimiterAllowsClassSizedSharedIPSuccesses(t *testing.T) {
	limiter, _ := testAuthRateLimiter(t, authRegistrationMode, 120, 3)
	app := newAuthTestApp(t, limiter, "/register/by-invite", true, "")

	for i := 0; i < 40; i++ {
		response := sendAuthRequest(t, app, "/register/by-invite", fmt.Sprintf("new-user-%02d", i), fmt.Sprintf("INVITE-%02d", i))
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("distinct registration %d was blocked with status %d", i, response.StatusCode)
		}
	}
}

func TestAuthRegistrationRateLimiterBlocksRepeatedFailuresPerIP(t *testing.T) {
	limiter, _ := testAuthRateLimiter(t, authRegistrationMode, 100, 3)
	app := newAuthTestApp(t, limiter, "/register/by-invite", false, "invalid or used invite_code")

	for i := 0; i < 3; i++ {
		response := sendAuthRequest(t, app, "/register/by-invite", fmt.Sprintf("guess-%d", i), fmt.Sprintf("BAD-%d", i))
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("registration failure %d unexpectedly blocked with status %d", i+1, response.StatusCode)
		}
	}
	response := sendAuthRequest(t, app, "/register/by-invite", "guess-beyond-cap", "BAD-BEYOND-CAP")
	if response.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("fourth registration failure status = %d, want %d", response.StatusCode, fiber.StatusTooManyRequests)
	}
}

func TestAuthRegistrationRateLimiterAllowsConcurrentClassroom(t *testing.T) {
	limiter, _ := testAuthRateLimiter(t, authRegistrationMode, 120, 3)
	app := fiber.New()
	entered := make(chan struct{}, 40)
	release := make(chan struct{})
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	app.Post("/register/by-invite", limiter, func(c *fiber.Ctx) error {
		entered <- struct{}{}
		<-release
		return c.JSON(map[string]any{"ok": true})
	})
	results := make(chan int, 40)
	for i := 0; i < 40; i++ {
		go func(i int) {
			request := httptest.NewRequest("POST", "/register/by-invite", strings.NewReader(fmt.Sprintf(`{"login":"student-%d","invite_code":"INVITE-%d"}`, i, i)))
			request.Header.Set("Content-Type", "application/json")
			response, err := app.Test(request, 5000)
			if err != nil {
				results <- 0
				return
			}
			defer response.Body.Close()
			results <- response.StatusCode
		}(i)
	}
	deadline := time.After(2 * time.Second)
	for i := 0; i < 40; i++ {
		select {
		case <-entered:
		case <-deadline:
			t.Fatalf("only %d of 40 simultaneous students reached registration", i)
		}
	}
	releaseOnce.Do(func() { close(release) })
	for i := 0; i < 40; i++ {
		if status := <-results; status != http.StatusOK {
			t.Fatalf("concurrent registration returned %d", status)
		}
	}
}
