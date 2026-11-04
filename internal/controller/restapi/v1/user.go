package v1

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/lucimux/arya/internal/controller/restapi/v1/request"
	"github.com/lucimux/arya/internal/controller/restapi/v1/response"
	"github.com/lucimux/arya/internal/entity"
)

// @Summary     Register
// @Description Register a new user
// @ID          register
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       request body     request.Register true "Registration data"
// @Success     201     {object} entity.User
// @Failure     400     {object} response.Error
// @Failure     409     {object} response.Error
// @Failure     500     {object} response.Error
// @Router      /auth/register [post]
func (r *V1) register(ctx *fiber.Ctx) error {
	var body request.Register

	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - register")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - register")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	role := entity.Role(body.Role)
	if !role.IsValid() {
		role = entity.RoleAdmin
	}

	user, err := r.u.Register(ctx.UserContext(), body.Username, body.Email, body.Password, role)
	if err != nil {
		r.l.Error(err, "restapi - v1 - register")

		if errors.Is(err, entity.ErrUserAlreadyExists) {
			return errorResponse(ctx, http.StatusConflict, "user already exists")
		}

		return errorResponse(ctx, http.StatusInternalServerError, "internal server error")
	}

	return ctx.Status(http.StatusCreated).JSON(user)
}

// @Summary     Login
// @Description Authenticate user and get JWT token
// @ID          login
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       request body     request.Login true "Login credentials"
// @Success     200     {object} response.Token
// @Failure     400     {object} response.Error
// @Failure     401     {object} response.Error
// @Failure     500     {object} response.Error
// @Router      /auth/login [post]
func (r *V1) login(ctx *fiber.Ctx) error {
	var body request.Login

	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - login")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - login")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	token, err := r.u.Login(ctx.UserContext(), body.Email, body.Password)
	if err != nil {
		r.l.Error(err, "restapi - v1 - login")

		if errors.Is(err, entity.ErrInvalidCredentials) {
			return errorResponse(ctx, http.StatusUnauthorized, "invalid credentials")
		}

		return errorResponse(ctx, http.StatusInternalServerError, "internal server error")
	}

	return ctx.Status(http.StatusOK).JSON(response.Token{Token: token})
}

// @Summary     Get profile
// @Description Get current user profile
// @ID          profile
// @Tags        user
// @Produce     json
// @Success     200 {object} entity.User
// @Failure     401 {object} response.Error
// @Failure     404 {object} response.Error
// @Failure     500 {object} response.Error
// @Security    BearerAuth
// @Router      /user/profile [get]
func (r *V1) profile(ctx *fiber.Ctx) error {
	userID, ok := ctx.Locals("userID").(string)
	if !ok {
		return errorResponse(ctx, http.StatusUnauthorized, "unauthorized")
	}

	user, err := r.u.GetUser(ctx.UserContext(), userID)
	if err != nil {
		r.l.Error(err, "restapi - v1 - profile")

		if errors.Is(err, entity.ErrUserNotFound) {
			return errorResponse(ctx, http.StatusNotFound, "user not found")
		}

		return errorResponse(ctx, http.StatusInternalServerError, "internal server error")
	}

	return ctx.Status(http.StatusOK).JSON(user)
}

// @Summary     List users
// @Description List all users (Super Admin only)
// @ID          list-users
// @Tags        user
// @Produce     json
// @Success     200 {array}  entity.User
// @Failure     401 {object} response.Error
// @Failure     403 {object} response.Error
// @Failure     500 {object} response.Error
// @Security    BearerAuth
// @Router      /users [get]
func (r *V1) listUsers(ctx *fiber.Ctx) error {
	users, err := r.u.List(ctx.UserContext())
	if err != nil {
		r.l.Error(err, "restapi - v1 - listUsers")

		return errorResponse(ctx, http.StatusInternalServerError, "internal server error")
	}

	return ctx.Status(http.StatusOK).JSON(users)
}

// @Summary     Update user role
// @Description Change user role (Super Admin only)
// @ID          update-user-role
// @Tags        user
// @Accept      json
// @Produce     json
// @Param       id      path     string                 true "User ID"
// @Param       request body     request.UpdateUserRole true "Role data"
// @Success     200     {object} map[string]string
// @Failure     400     {object} response.Error
// @Failure     401     {object} response.Error
// @Failure     403     {object} response.Error
// @Failure     404     {object} response.Error
// @Failure     500     {object} response.Error
// @Security    BearerAuth
// @Router      /users/{id}/role [put]
func (r *V1) updateUserRole(ctx *fiber.Ctx) error {
	id := ctx.Params("id")

	var body request.UpdateUserRole
	if err := ctx.BodyParser(&body); err != nil {
		r.l.Error(err, "restapi - v1 - updateUserRole")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	if err := r.v.Struct(body); err != nil {
		r.l.Error(err, "restapi - v1 - updateUserRole")

		return errorResponse(ctx, http.StatusBadRequest, "invalid request body")
	}

	role := entity.Role(body.Role)
	if !role.IsValid() {
		return errorResponse(ctx, http.StatusBadRequest, "invalid role")
	}

	if err := r.u.UpdateRole(ctx.UserContext(), id, role); err != nil {
		r.l.Error(err, "restapi - v1 - updateUserRole")

		if errors.Is(err, entity.ErrUserNotFound) {
			return errorResponse(ctx, http.StatusNotFound, "user not found")
		}

		return errorResponse(ctx, http.StatusInternalServerError, "internal server error")
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"status": "ok"})
}
