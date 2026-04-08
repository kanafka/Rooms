package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"room-booking/internal/domain"
)

// handleDummyLogin godoc
// @Summary      Получить тестовый JWT по роли
// @Description  Выдаёт тестовый JWT для роли admin или user. Фиксированный UUID для каждой роли.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body object{role=string} true "Роль (admin или user)"
// @Success      200 {object} object{token=string}
// @Failure      400 {object} object{error=object{code=string,message=string}}
// @Failure      500 {object} object{error=object{code=string,message=string}}
// @Router       /dummyLogin [post]
func handleDummyLogin(authUC AuthUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Role string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
			return
		}

		token, err := authUC.DummyLogin(req.Role)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidRequest) {
				writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"token": token})
	}
}

// handleRegister godoc
// @Summary      Регистрация пользователя (дополнительное задание)
// @Description  Создаёт нового пользователя с email, паролем и ролью.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body object{email=string,password=string,role=string} true "Данные пользователя"
// @Success      201 {object} object{user=domain.User}
// @Failure      400 {object} object{error=object{code=string,message=string}}
// @Failure      500 {object} object{error=object{code=string,message=string}}
// @Router       /register [post]
func handleRegister(authUC AuthUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
			return
		}

		if req.Email == "" || req.Password == "" || req.Role == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "email, password and role are required")
			return
		}

		user, err := authUC.Register(r.Context(), req.Email, req.Password, req.Role)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrEmailTaken):
				writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "email already taken")
			case errors.Is(err, domain.ErrInvalidRequest):
				writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
			default:
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			}
			return
		}

		writeJSON(w, http.StatusCreated, map[string]interface{}{"user": user})
	}
}

// handleLogin godoc
// @Summary      Авторизация по email и паролю (дополнительное задание)
// @Description  Авторизует пользователя, возвращает JWT.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body object{email=string,password=string} true "Учётные данные"
// @Success      200 {object} object{token=string}
// @Failure      401 {object} object{error=object{code=string,message=string}}
// @Failure      500 {object} object{error=object{code=string,message=string}}
// @Router       /login [post]
func handleLogin(authUC AuthUsecase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
			return
		}

		if req.Email == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "email and password are required")
			return
		}

		token, err := authUC.Login(r.Context(), req.Email, req.Password)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidCredentials) {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
				return
			}
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"token": token})
	}
}
