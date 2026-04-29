package domain

type Role string

const (
	RoleOwner        Role = "owner"
	RoleReceptionist Role = "receptionist"
	RoleTeacher      Role = "teacher"
	RoleStudent      Role = "student"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleReceptionist, RoleTeacher, RoleStudent:
		return true
	default:
		return false
	}
}

func (r Role) RequiresBranchScope() bool {
	return r != RoleOwner
}

func (r Role) CanApproveTeacherSalary() bool {
	return r == RoleOwner
}

func (r Role) CanViewTeacherSalary(subjectUserID, viewerUserID string) bool {
	if r == RoleOwner {
		return true
	}

	return r == RoleTeacher && subjectUserID != "" && subjectUserID == viewerUserID
}
