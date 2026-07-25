package helper

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

func HumanValidationErrors(err error) []string {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return []string{"Body inválido"}
	}

	msgs := make([]string, 0, len(ve))
	for _, fe := range ve {
		field := strings.ToLower(fe.Field())

		switch fe.Tag() {
		case "required":
			msgs = append(msgs, fmt.Sprintf("El campo %s es obligatorio", field))
		case "email":
			msgs = append(msgs, fmt.Sprintf("El campo %s debe ser un correo válido", field))
		case "min":
			msgs = append(msgs, fmt.Sprintf("El campo %s debe tener al menos %s caracteres", field, fe.Param()))
		case "max":
			msgs = append(msgs, fmt.Sprintf("El campo %s debe tener máximo %s caracteres", field, fe.Param()))
		default:
			msgs = append(msgs, fmt.Sprintf("El campo %s no es válido", field))
		}
	}

	return msgs
}
