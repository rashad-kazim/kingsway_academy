package domain

import (
	"errors"
	"testing"
	"time"
)

func TestParseFIN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    FIN
		wantErr bool
	}{
		{name: "normalizes lowercase", value: " a1b2c3d ", want: FIN("A1B2C3D")},
		{name: "rejects short value", value: "A1B2C3", wantErr: true},
		{name: "rejects symbols", value: "A1B-3D4", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseFIN(tt.value)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidFIN) {
					t.Fatalf("expected ErrInvalidFIN, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestRolePermissions(t *testing.T) {
	t.Parallel()

	if !RoleReceptionist.RequiresBranchScope() {
		t.Fatal("receptionist must be branch scoped")
	}
	if RoleOwner.RequiresBranchScope() {
		t.Fatal("owner should not require one fixed branch scope")
	}
	if RoleReceptionist.CanViewTeacherSalary("teacher-1", "receptionist-1") {
		t.Fatal("receptionist must not view teacher salary")
	}
	if !RoleTeacher.CanViewTeacherSalary("teacher-1", "teacher-1") {
		t.Fatal("teacher should see own salary")
	}
}

func TestRetentionUntil(t *testing.T) {
	t.Parallel()

	uploadedAt := time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)
	retention := RetentionUntil(FilePurposeExamWriting, uploadedAt)
	if retention == nil {
		t.Fatal("expected writing retention date")
	}
	if got, want := retention.Sub(uploadedAt), WritingRetention; got != want {
		t.Fatalf("expected %s retention, got %s", want, got)
	}

	if RetentionUntil(FilePurposePassport, uploadedAt) != nil {
		t.Fatal("passport retention should not default to writing cleanup")
	}
}

func TestCalculateSwapAllocation(t *testing.T) {
	t.Parallel()

	first, second := CalculateSwapAllocation(30000, 12, 18)
	if first != 12000 || second != 18000 {
		t.Fatalf("expected 12000/18000, got %d/%d", first, second)
	}
}

func TestTimeRangeOverlaps(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)
	first := TimeRange{Start: base, End: base.Add(time.Hour)}
	overlap := TimeRange{Start: base.Add(30 * time.Minute), End: base.Add(90 * time.Minute)}
	adjacent := TimeRange{Start: base.Add(time.Hour), End: base.Add(2 * time.Hour)}

	if !first.Overlaps(overlap) {
		t.Fatal("expected overlapping time ranges")
	}
	if first.Overlaps(adjacent) {
		t.Fatal("adjacent time ranges should not overlap")
	}
}
