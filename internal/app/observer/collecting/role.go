package collecting

// Role type of membership role
type Role int

//go:generate stringer -type=Role
const (
	RoleChairMan Role = iota + 1
	RoleTreasure
	RoleMember
)
