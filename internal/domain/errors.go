package domain

import "errors"

var (
	ErrNotFound           = errors.New("Не найдено")
	ErrConflict           = errors.New("Конфликт")
	ErrForbidden          = errors.New("Нет доступа")
	ErrUnauthorized       = errors.New("Не авторизован")
	ErrInvalidCredentials = errors.New("Неверные логин или пароль")
	ErrValidation         = errors.New("Неверные данные")
)
