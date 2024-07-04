package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dgrijalva/jwt-go"
)

var (
	cognitoUserPoolIssuer = "your_user_pool_url"
	cognitoIdentityIssuer = "your_identity_pool_url"
	audience              = "your_client_id"
)

type TokenClaims struct {
	jwt.StandardClaims
	Scope string `json:"scope"`
}

type CustomAuthorizerResponse struct {
	PrincipalID string                                  `json:"principalId"`
	Policy      events.APIGatewayCustomAuthorizerPolicy `json:"policyDocument"`
	Context     map[string]interface{}                  `json:"context"`
}

func Handler(ctx context.Context, event events.APIGatewayCustomAuthorizerRequest) (CustomAuthorizerResponse, error) {
	token := event.AuthorizationToken
	if token == "" {
		return CustomAuthorizerResponse{}, errors.New("missing token")
	}

	if strings.HasPrefix(token, "Bearer ") {
		token = strings.TrimPrefix(token, "Bearer ")
	}

	claims := &TokenClaims{}
	var issuer string

	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		issuer = token.Claims.(jwt.MapClaims)["iss"].(string)
		switch issuer {
		case cognitoUserPoolIssuer:
			return getPublicKey(token)
		case cognitoIdentityIssuer:
			return nil, nil
		default:
			return nil, fmt.Errorf("unknown issuer: %s", issuer)
		}
	})

	if err != nil {
		return CustomAuthorizerResponse{}, fmt.Errorf("invalid token: %v", err)
	}

	if issuer == cognitoUserPoolIssuer && claims.Audience == audience {
		return generatePolicy("user", "Allow", event.MethodArn), nil
	} else if issuer == cognitoIdentityIssuer {
		return generatePolicy("user", "Allow", event.MethodArn), nil
	}

	return CustomAuthorizerResponse{}, errors.New("invalid token or issuer")
}

func generatePolicy(principalID, effect, resource string) CustomAuthorizerResponse {
	policy := CustomAuthorizerResponse{
		PrincipalID: principalID,
		Policy: events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   effect,
					Resource: []string{resource},
				},
			},
		},
	}

	return policy
}

func getPublicKey(token *jwt.Token) (interface{}, error) {
	return nil, errors.New("public key not implemented")
}

func main() {
	lambda.Start(Handler)
}
