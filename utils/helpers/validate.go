package helpers

import (
	// Local Packages
	errors "flowx/errors"
)

func ValidateRequiredString(ve *errors.ValidationErrorBuilder, field, value string) {
	if value == "" {
		ve.Add(field, "cannot be empty")
	}
}

func ValidateRequiredNumeric(ve *errors.ValidationErrorBuilder, field string, value int) {
	if value == 0 {
		ve.Add(field, "cannot be empty")
	}
}
