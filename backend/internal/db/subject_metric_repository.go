package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubjectMetricRepository struct {
	pool *pgxpool.Pool
}

func NewSubjectMetricRepository(pool *pgxpool.Pool) *SubjectMetricRepository {
	return &SubjectMetricRepository{pool: pool}
}

func (r *SubjectMetricRepository) Upsert(ctx context.Context, metric SubjectMetric) (SubjectMetric, error) {
	var out SubjectMetric
	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO subject_metrics (
			subject_id,
			zet_expert,
			zet_fact,
			hours_expert,
			hours_by_plan,
			hours_contr_work,
			hours_auditory,
			hours_self_study,
			hours_control,
			hours_prep
		 ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 ON CONFLICT (subject_id) DO UPDATE SET
			zet_expert = EXCLUDED.zet_expert,
			zet_fact = EXCLUDED.zet_fact,
			hours_expert = EXCLUDED.hours_expert,
			hours_by_plan = EXCLUDED.hours_by_plan,
			hours_contr_work = EXCLUDED.hours_contr_work,
			hours_auditory = EXCLUDED.hours_auditory,
			hours_self_study = EXCLUDED.hours_self_study,
			hours_control = EXCLUDED.hours_control,
			hours_prep = EXCLUDED.hours_prep
		 RETURNING subject_id, zet_expert, zet_fact, hours_expert, hours_by_plan,
		 	hours_contr_work, hours_auditory, hours_self_study, hours_control, hours_prep`,
		metric.SubjectID,
		metric.ZetExpert,
		metric.ZetFact,
		metric.HoursExpert,
		metric.HoursByPlan,
		metric.HoursContrWork,
		metric.HoursAuditory,
		metric.HoursSelfStudy,
		metric.HoursControl,
		metric.HoursPrep,
	).Scan(
		&out.SubjectID,
		&out.ZetExpert,
		&out.ZetFact,
		&out.HoursExpert,
		&out.HoursByPlan,
		&out.HoursContrWork,
		&out.HoursAuditory,
		&out.HoursSelfStudy,
		&out.HoursControl,
		&out.HoursPrep,
	)
	if err != nil {
		return SubjectMetric{}, fmt.Errorf("upsert subject metric: %w", err)
	}

	return out, nil
}

func (r *SubjectMetricRepository) GetBySubjectID(ctx context.Context, subjectID int32) (SubjectMetric, bool, error) {
	var out SubjectMetric
	err := r.pool.QueryRow(
		ctx,
		`SELECT subject_id, zet_expert, zet_fact, hours_expert, hours_by_plan,
		 	hours_contr_work, hours_auditory, hours_self_study, hours_control, hours_prep
		 FROM subject_metrics
		 WHERE subject_id = $1`,
		subjectID,
	).Scan(
		&out.SubjectID,
		&out.ZetExpert,
		&out.ZetFact,
		&out.HoursExpert,
		&out.HoursByPlan,
		&out.HoursContrWork,
		&out.HoursAuditory,
		&out.HoursSelfStudy,
		&out.HoursControl,
		&out.HoursPrep,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return SubjectMetric{}, false, nil
	}
	if err != nil {
		return SubjectMetric{}, false, fmt.Errorf("get subject metric by subject id: %w", err)
	}

	return out, true, nil
}

func (r *SubjectMetricRepository) DeleteBySubjectID(ctx context.Context, subjectID int32) (bool, error) {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM subject_metrics WHERE subject_id = $1`, subjectID)
	if err != nil {
		return false, fmt.Errorf("delete subject metric: %w", err)
	}
	return cmd.RowsAffected() > 0, nil
}
