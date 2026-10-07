package adapter_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/g04/backend/internal/adapter/repository"
	"github.com/g04/backend/internal/domain"
)

// Scripted SQL transport: tests error propagation and decoding, not SQL aggregation.
type dashboardSQLStep struct {
	columns []string
	values  [][]driver.Value
	err     error
	rowErr  error
}
type dashboardConnector struct {
	steps []dashboardSQLStep
	next  int
}

func (c *dashboardConnector) Connect(context.Context) (driver.Conn, error) {
	return &dashboardConn{c}, nil
}
func (c *dashboardConnector) Driver() driver.Driver { return dashboardDriver{} }

type dashboardDriver struct{}

func (dashboardDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use connector") }

type dashboardConn struct{ c *dashboardConnector }

func (*dashboardConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not supported") }
func (*dashboardConn) Close() error                        { return nil }
func (*dashboardConn) Begin() (driver.Tx, error)           { return nil, errors.New("not supported") }
func (c *dashboardConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	if c.c.next >= len(c.c.steps) {
		return nil, errors.New("unexpected query")
	}
	s := c.c.steps[c.c.next]
	c.c.next++
	if s.err != nil {
		return nil, s.err
	}
	return &dashboardRows{step: s}, nil
}

type dashboardRows struct {
	step dashboardSQLStep
	next int
}

func (r *dashboardRows) Columns() []string { return r.step.columns }
func (*dashboardRows) Close() error        { return nil }
func (r *dashboardRows) Next(dest []driver.Value) error {
	if r.next >= len(r.step.values) {
		if r.step.rowErr != nil {
			return r.step.rowErr
		}
		return io.EOF
	}
	copy(dest, r.step.values[r.next])
	r.next++
	return nil
}
func dashboardSteps() []dashboardSQLStep {
	return []dashboardSQLStep{
		{columns: []string{"count"}, values: [][]driver.Value{{int64(10)}}},
		{columns: []string{"risk"}, values: [][]driver.Value{{int64(4)}}},
		{columns: []string{"pending"}, values: [][]driver.Value{{int64(3)}}},
		{columns: []string{"completed"}, values: [][]driver.Value{{int64(2)}}},
		{columns: []string{"station", "count"}, values: [][]driver.Value{{"FAL", int64(3)}, {"PIN", int64(1)}}},
	}
}
func dashboardDB(t *testing.T, steps []dashboardSQLStep) *sql.DB {
	t.Helper()
	db := sql.OpenDB(&dashboardConnector{steps: steps})
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRF10_RepositorySummaryValues(t *testing.T) {
	for _, empty := range []bool{false, true} {
		name := "populated"
		if empty {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			steps := dashboardSteps()
			want := &domain.DashboardSummary{TotalDoors: 10, DoorsAtRisk: 4, PendingInspections: 3, CompletedInspections: 2, CriticalStations: []domain.StationRiskSummary{{StationCode: "FAL", DoorsAtRisk: 3}, {StationCode: "PIN", DoorsAtRisk: 1}}}
			if empty {
				for i := 0; i < 4; i++ {
					steps[i].values = [][]driver.Value{{int64(0)}}
				}
				steps[1].values = nil // No prediction run is legitimate, not a database failure.
				steps[4].values = nil
				want = &domain.DashboardSummary{}
			}
			got, err := repository.NewPostgresDashboardRepository(dashboardDB(t, steps)).GetSummary(context.Background())
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("summary=%+v err=%v, want=%+v", got, err, want)
			}
		})
	}
}

func TestRF10_RepositoryRejectsIncompleteSummary(t *testing.T) {
	for index, name := range []string{"doors_query", "risk_query", "pending_query", "completed_query", "ranking_query", "ranking_scan", "ranking_iteration"} {
		t.Run(name, func(t *testing.T) {
			steps := dashboardSteps()
			sentinel := errors.New("database unavailable")
			if index < 5 {
				steps[index].err = sentinel
			}
			if index == 5 {
				steps[4].values = [][]driver.Value{{"FAL", "invalid count"}}
			}
			if index == 6 {
				steps[4].rowErr = sentinel
			}
			got, err := repository.NewPostgresDashboardRepository(dashboardDB(t, steps)).GetSummary(context.Background())
			if err == nil || got != nil {
				t.Fatalf("partial dashboard accepted: %+v err=%v", got, err)
			}
		})
	}
}
