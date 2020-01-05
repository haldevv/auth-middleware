package v1

import (
	v1 "auth-middleware/pkg/api/v1"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"google.golang.org/api/oauth2/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"time"

	"net/http"
)

const (
	apiVersion = "v1"
)

var httpClient = &http.Client{}

type authServiceServer struct {
  options *redis.Options
}

func NewAuthServiceServer (conf *redis.Options) v1.AuthServer {
	return &authServiceServer{options: conf}
}

func (authServer *authServiceServer) Validate (ctx context.Context, req *v1.MessageRequest) (*v1.MessageResponse, error)  {
	if err := authServer.CheckAPI(req.Api); err != nil {
		return nil, err
	}

	// creo un cliente de redis
	client := redis.NewClient(authServer.options)
	defer client.Close()

	// Reviso si tengo almacenado el usuario
	data, err :=  client.Get(req.Email).Result()
	if err != nil || err == redis.Nil {
		validationResponse := verifyIdToken(req.Token)
		if validationResponse.Stats == v1.Status_VALID {
			go saveUser(validationResponse, authServer.options)
		}
		return validationResponse, nil
	}
	var userData v1.MessageResponse
	err = json.Unmarshal([]byte(data), &userData)
	if strings.Compare(req.Token, userData.Token) != 0 {
		println("Token is no the same")
		validationResponse := verifyIdToken(req.Token)
		if validationResponse.Stats == v1.Status_VALID {
			validationResponse.Token = req.Token
			go saveUser(validationResponse, authServer.options)
		}
	}
	response := v1.MessageResponse{
		Stats:                v1.Status_VALID,
		Message:              "Obtained from store",
		Email:                userData.Email,
		UserId:               userData.UserId,
		Api:                  req.Api,
		VerifiedEmail:        userData.VerifiedEmail,
		Token:                userData.Token,
	}
	return &response, nil
}

func (authServer *authServiceServer) CheckAPI(api string) error  {
	if len(api) > 0 {
		if apiVersion != api {
			return status.Errorf(codes.Unimplemented, "unsupported API version: service implements API version '%s', but asked for '%s'", apiVersion, api)
		}
	}
	return nil
}

func verifyIdToken(idToken string) *v1.MessageResponse {
	oauth2Service, _ := oauth2.New(httpClient)
	tokenInfoCall := oauth2Service.Tokeninfo()
	tokenInfoCall.IdToken(idToken)
	tokenInfo, requestError := tokenInfoCall.Do()
	if requestError != nil {
		return &v1.MessageResponse{
			Stats:                v1.Status_INVALID,
			Message:			  "Token Invalid",
		}
	}
	return &v1.MessageResponse {
		Stats:                v1.Status_VALID,
		Message:              "Obtained from Google Servers",
		Email:                tokenInfo.Email,
		UserId:               tokenInfo.UserId,
		VerifiedEmail:        tokenInfo.VerifiedEmail,
	}
}

func saveUser(data *v1.MessageResponse, options *redis.Options) {
	client := redis.NewClient(options)
	defer client.Close()
	cacheEntry, _ := json.Marshal(data)
	expiration, _ := time.ParseDuration("6h")
	saveErr := client.Set(data.Email, cacheEntry, expiration).Err()
	if saveErr != nil {
		fmt.Println("Error to save userData", saveErr)
	}
	fmt.Println("New Token Saved")
}
