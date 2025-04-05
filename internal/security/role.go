package security

type Role string
type Permission string

const (
	Admin        Role = "admin"
	Contractor   Role = "contractor"
	Manufacturer Role = "manufacturer"
	Designer     Role = "designer"
	HomeOwner    Role = "homeowner"
)

var ValidRole = map[Role]bool{
	Admin:        true,
	Contractor:   true,
	Manufacturer: true,
	Designer:     true,
	HomeOwner:    true,
}

const (
	PermAdmin        Permission = "admin"
	PermUser         Permission = "user"
	PermContractor   Permission = "contractor"
	PermManufacturer Permission = "manufacturer"
	PermDesigner     Permission = "designer"
	PermHomeOwner    Permission = "homeowner"
)

var RolePermissions = map[Role]map[Permission]bool{
	Admin: {
		PermAdmin:        true,
		PermUser:         true,
		PermContractor:   true,
		PermManufacturer: true,
		PermDesigner:     true,
		PermHomeOwner:    true,
	},
	Contractor: {
		PermUser:       true,
		PermContractor: true,
	},
	Manufacturer: {
		PermUser:         true,
		PermManufacturer: true,
	},
	Designer: {
		PermUser:     true,
		PermDesigner: true,
	},
	HomeOwner: {
		PermUser:      true,
		PermHomeOwner: true,
	},
}

func IsValid(role string) bool {
	_, ok := ValidRole[Role(role)]
	return ok
}

func HasPermission(role Role, perm Permission) bool {
	_, ok := RolePermissions[role][perm]
	return ok
}
