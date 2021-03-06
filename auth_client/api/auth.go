package api

import (
	"auth_client/errs"
	"auth_client/proto"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi"
)

type registerInput struct {
	Fname        string `json:"fname"`
	Lname        string `json:"lname"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	Organisation string `json:"organisation"`
}

type updateUserInput struct {
	Fname        string `json:"fname"`
	Lname        string `json:"lname"`
	Organisation string `json:"organisation"`
	AccessToken  string `json:"access_token"`
}

type authInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type resetPasswordInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Token    string `json:"token"`
}
type respTokens struct {
	AccessToken   string `json:"access_token"`
	RefreshToken  string `json:"refresh_token"`
	AccessExpires int64  `json:"access_expires"`
}

type userData struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	Admin bool   `json:"admin"`
}
type Secret struct {
	SecretKey  string
	ExpireDate string
	CreatedAt  string
}
type SecretsList struct {
	Secrets []Secret
}

func (a *Api) register(w http.ResponseWriter, r *http.Request) error {
	var input registerInput

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return &errs.ApiError{Code: http.StatusInternalServerError, Message: err.Error()}
	}
	if err := json.Unmarshal(body, &input); err != nil {
		return &errs.ApiError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	if err := input.validate(); err != nil {
		return &errs.ApiError{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("validation err: %v", err),
		}
	}

	ctx := context.Background()
	tokens, err := a.AuthGRPC.Register(ctx, &proto.RegisterUserData{
		Fname:        input.Fname,
		Lname:        input.Lname,
		Email:        input.Email,
		Password:     input.Password,
		Organisation: input.Organisation,
	})
	if err != nil {
		return err
	}

	respTokens := respTokens{
		AccessToken:   tokens.AccessToken,
		RefreshToken:  tokens.RefreshToken,
		AccessExpires: tokens.AccessExpires,
	}
	json.NewEncoder(w).Encode(respTokens)
	return nil
}

func (a *Api) login(w http.ResponseWriter, r *http.Request) error {
	var input authInput
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return &errs.ApiError{Code: http.StatusInternalServerError, Message: err.Error()}
	}
	if err := json.Unmarshal(body, &input); err != nil {
		return &errs.ApiError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	ctx := context.Background()
	tokens, err := a.AuthGRPC.Login(ctx, &proto.ReqUserData{
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		return err
	}

	respTokens := respTokens{
		AccessToken:   tokens.AccessToken,
		RefreshToken:  tokens.RefreshToken,
		AccessExpires: tokens.AccessExpires,
	}
	json.NewEncoder(w).Encode(respTokens)
	return nil
}

func (a *Api) profile(w http.ResponseWriter, r *http.Request) error {
	token, err := tokenFromHeader(r)
	if err != nil {
		return &errs.ApiError{Code: http.StatusUnauthorized, Message: err.Error()}
	}

	ctx := context.Background()
	data, err := a.AuthGRPC.Profile(ctx, &proto.AccessToken{
		AccessToken: token,
	})
	if err != nil {
		return err
	}

	userData := userData{
		ID:    uint(data.Id),
		Email: data.Email,
		Admin: data.Admin,
	}
	json.NewEncoder(w).Encode(userData)
	return nil
}
func (a *Api) profileDelete(w http.ResponseWriter, r *http.Request) error {
	token, err := tokenFromHeader(r)
	if err != nil {
		return &errs.ApiError{Code: http.StatusUnauthorized, Message: err.Error()}
	}
	ctx := context.Background()
	data, err := a.AuthGRPC.ProfileDelete(ctx, &proto.AccessToken{
		AccessToken: token,
	})
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(data)
	return nil
}
func (a *Api) profileUpdate(w http.ResponseWriter, r *http.Request) error {
	var updateInput updateUserInput
	token, err := tokenFromHeader(r)
	if err != nil {
		return &errs.ApiError{Code: http.StatusUnauthorized, Message: err.Error()}
	}
	body, err := ioutil.ReadAll(r.Body)

	if err := json.Unmarshal(body, &updateInput); err != nil {
		return &errs.ApiError{Code: http.StatusBadRequest, Message: err.Error()}
	}
	ctx := context.Background()
	data, err := a.AuthGRPC.ProfileUpdate(ctx, &proto.UpdateUserData{
		Fname:        updateInput.Fname,
		Lname:        updateInput.Lname,
		Organisation: updateInput.Organisation,
		AccessToken: &proto.AccessToken{
			AccessToken: token,
		},
	})
	if err != nil {
		return err
	}

	userData := updateUserInput{
		Fname:        data.Fname,
		Lname:        data.Lname,
		Organisation: data.Organisation,
	}
	json.NewEncoder(w).Encode(userData)
	return nil
}
func (a *Api) profileList(w http.ResponseWriter, r *http.Request) error {
	token, err := tokenFromHeader(r)
	if err != nil {
		return &errs.ApiError{Code: http.StatusUnauthorized, Message: err.Error()}
	}

	ctx := context.Background()
	data, err := a.AuthGRPC.ProfilesList(ctx, &proto.AccessToken{
		AccessToken: token,
	})
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(data)
	return nil
}
func (a *Api) createSecret(w http.ResponseWriter, r *http.Request) error {
	token, err := tokenFromHeader(r)
	if err != nil {
		return &errs.ApiError{Code: http.StatusUnauthorized, Message: err.Error()}
	}

	ctx := context.Background()
	data, err := a.AuthGRPC.CreateSecret(ctx, &proto.AccessToken{
		AccessToken: token,
	})
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(data)
	return nil
}
func (a *Api) getSecrets(w http.ResponseWriter, r *http.Request) error {
	token, err := tokenFromHeader(r)
	if err != nil {
		return &errs.ApiError{Code: http.StatusUnauthorized, Message: err.Error()}
	}

	ctx := context.Background()
	data, err := a.AuthGRPC.GetSecrets(ctx, &proto.AccessToken{
		AccessToken: token,
	})
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(data)
	return nil
}
func (a *Api) deleteSecrets(w http.ResponseWriter, r *http.Request) error {
	token, err := tokenFromHeader(r)
	if err != nil {
		return &errs.ApiError{Code: http.StatusUnauthorized, Message: err.Error()}
	}

	ctx := context.Background()
	sid, _ := strconv.Atoi(chi.URLParam(r, "id"))

	data, err := a.AuthGRPC.DeleteSecret(ctx, &proto.ReqDeleteSecret{
		SecretId: int32(sid),
		AccessToken: &proto.AccessToken{
			AccessToken: token,
		},
	})
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(data)
	return nil
}

func (a *Api) getSecretExpired(w http.ResponseWriter, r *http.Request) error {
	token, err := tokenFromHeader(r)
	if err != nil {
		return &errs.ApiError{Code: http.StatusUnauthorized, Message: err.Error()}
	}

	ctx := context.Background()
	sid, _ := strconv.Atoi(chi.URLParam(r, "id"))

	data, err := a.AuthGRPC.GetSecret(ctx, &proto.ReqGetSecretExpire{
		SecretId: int32(sid),
		AccessToken: &proto.AccessToken{
			AccessToken: token,
		},
	})
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(data)
	return nil
}

func (a *Api) forgotPassword(w http.ResponseWriter, r *http.Request) error {
	var input authInput
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return &errs.ApiError{Code: http.StatusInternalServerError, Message: err.Error()}
	}
	if err := json.Unmarshal(body, &input); err != nil {
		return &errs.ApiError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	ctx := context.Background()
	resp, err := a.AuthGRPC.ForgotPassword(ctx, &proto.ReqUserData{
		Email: input.Email,
	})
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(resp)
	return nil
}
func (a *Api) resetPassword(w http.ResponseWriter, r *http.Request) error {
	var input resetPasswordInput
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return &errs.ApiError{Code: http.StatusInternalServerError, Message: err.Error()}
	}
	if err := json.Unmarshal(body, &input); err != nil {
		return &errs.ApiError{Code: http.StatusBadRequest, Message: err.Error()}
	}

	ctx := context.Background()
	resp, err := a.AuthGRPC.ResetPassword(ctx, &proto.ReqResetPassword{
		Email:    input.Email,
		Password: input.Password,
		Token:    input.Token,
	})
	if err != nil {
		return err
	}

	json.NewEncoder(w).Encode(resp)
	return nil
}

func (a *Api) refreshTokens(w http.ResponseWriter, r *http.Request) error {
	token, err := tokenFromHeader(r)
	if err != nil {
		return &errs.ApiError{Code: http.StatusUnauthorized, Message: err.Error()}
	}

	ctx := context.Background()
	tokens, err := a.AuthGRPC.RefreshTokens(ctx, &proto.RefreshToken{
		RefreshToken: token,
	})
	if err != nil {
		return err
	}

	respTokens := respTokens{
		AccessToken:   tokens.AccessToken,
		RefreshToken:  tokens.RefreshToken,
		AccessExpires: tokens.AccessExpires,
	}
	json.NewEncoder(w).Encode(respTokens)
	return nil
}

func tokenFromHeader(r *http.Request) (string, error) {
	bearer := r.Header.Get("Authorization")
	if len(bearer) > 7 && strings.ToUpper(bearer[0:6]) == "BEARER" {
		return bearer[7:], nil
	}
	return "", fmt.Errorf("jwt token not found or wrong structure")
}

var emailRegexp = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

func (a authInput) validate() error {

	if a.Password == "" {
		return fmt.Errorf("password is required")
	}
	if utf8.RuneCountInString(a.Password) < 8 || utf8.RuneCountInString(a.Email) > 40 {
		return fmt.Errorf("password must be from 8 to 40 characters")
	}

	if a.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !emailRegexp.MatchString(a.Email) {
		return fmt.Errorf("email is not valid")
	}

	return nil
}

func (a registerInput) validate() error {

	if a.Password == "" {
		return fmt.Errorf("password is required")
	}
	if utf8.RuneCountInString(a.Password) < 8 || utf8.RuneCountInString(a.Email) > 40 {
		return fmt.Errorf("password must be from 8 to 40 characters")
	}

	if a.Email == "" {
		return fmt.Errorf("email is required")
	}
	if !emailRegexp.MatchString(a.Email) {
		return fmt.Errorf("email is not valid")
	}

	return nil
}
