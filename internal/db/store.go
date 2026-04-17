package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool

	Users           *UserRepository
	Lecterns        *LecternRepository
	ControlTypes    *ControlTypeRepository
	Subjects        *SubjectRepository
	SubjectMetrics  *SubjectMetricRepository
	SemesterLoads   *SemesterLoadRepository
	SubjectControls *SubjectControlRepository
	Teachers        *TeacherRepository
	Groups          *GroupRepository
	Students        *StudentRepository
	Attendance      *AttendanceRepository
}

func NewStore(ctx context.Context, dsn string) (*Store, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, fmt.Errorf("empty database dsn")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	store := &Store{
		pool:            pool,
		Users:           NewUserRepository(pool),
		Lecterns:        NewLecternRepository(pool),
		ControlTypes:    NewControlTypeRepository(pool),
		Subjects:        NewSubjectRepository(pool),
		SubjectMetrics:  NewSubjectMetricRepository(pool),
		SemesterLoads:   NewSemesterLoadRepository(pool),
		SubjectControls: NewSubjectControlRepository(pool),
		Teachers:        NewTeacherRepository(pool),
		Groups:          NewGroupRepository(pool),
		Students:        NewStudentRepository(pool),
		Attendance:      NewAttendanceRepository(pool),
	}

	return store, nil
}

func (s *Store) Close() {
	if s == nil || s.pool == nil {
		return
	}
	s.pool.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("store is not initialized")
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.pool.Ping(pingCtx); err != nil {
		return fmt.Errorf("postgres ping failed: %w", err)
	}
	return nil
}

func (s *Store) Pool() *pgxpool.Pool {
	if s == nil {
		return nil
	}
	return s.pool
}
