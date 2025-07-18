package main

import (
	"github.com/hnzhou16/project-cocraft-server/internal/security"
	"reflect"
	"regexp"

	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

// init before main function
func init() {
	Validate = validator.New(validator.WithRequiredStructEnabled())
	_ = Validate.RegisterValidation("valid_email", ValidateEmail)
	_ = Validate.RegisterValidation("valid_password", ValidatePassword)
	_ = Validate.RegisterValidation("valid_role", ValidateRole)
	_ = Validate.RegisterValidation("valid_roles_slice", ValidateRoleSlice)
}

func ValidateEmail(fl validator.FieldLevel) bool {
	email := fl.Field().String()

	regex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return regex.MatchString(email)
}

func ValidatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	if len(password) < 8 {
		return false
	}

	var upper, lower, digit int

	for _, ch := range password {
		switch {
		case ch >= 'A' && ch <= 'Z':
			upper++
		case ch >= 'a' && ch <= 'z':
			lower++
		case ch >= '0' && ch <= '9':
			digit++
		}
	}

	return upper > 0 && lower > 0 && digit > 0
}

func ValidateRole(fl validator.FieldLevel) bool {
	role := fl.Field().String()
	return security.IsValid(role)
}

func ValidateRoleSlice(fl validator.FieldLevel) bool {
	field := fl.Field()

	if field.Kind() != reflect.Slice {
		return false
	}

	for i := 0; i < field.Len(); i++ {
		elem := field.Index(i)

		// Convert the element to interface{} and assert it's a string-like value
		role, ok := elem.Interface().(security.Role)
		if !ok {
			return false
		}

		if !security.ValidRole[role] {
			return false
		}
	}

	return true
}
