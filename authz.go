package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"log"

	"github.com/dgrijalva/jwt-go"
)

const googleApisCertsURL = "https://www.googleapis.com/oauth2/v1/certs"

type GoogleClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	FirstName     string `json:"given_name"`
	LastName      string `json:"family_name"`
	jwt.StandardClaims
}

func getGoogleSigningPubKey(keyID string) (string, error) {
	resp, err := http.Get(googleApisCertsURL)
	if err != nil {
		return "", err
	}
	log.Println(resp.Body)
	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("readAll failed")
		return "", err
	}

	myResp := map[string]string{}
	err = json.Unmarshal(dat, &myResp)
	if err != nil {
		log.Println("Unmarshal failed")
		return "", err
	}
	key, ok := myResp[keyID]
	if !ok {
		return "", errors.New("key not found")
	}

	return key, nil
}

func ValidateGoogleJWT(tokenString string) (GoogleClaims, error) {
	claimsStruct := GoogleClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claimsStruct,
		func(token *jwt.Token) (interface{}, error) {
			pem, err := getGoogleSigningPubKey(token.Header["kid"])
			if err != nil {
				return nil, err
			}
			log.Println(pem)
			key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pem))
			if err != nil {
				log.Println("Error parsing public key from certificate")
				return nil, err
			}
			return key, nil
		},
	)
	if err != nil {
		log.Println(err)
		return GoogleClaims{}, err
	}

	claims, ok := token.Claims.(*GoogleClaims)
	if !ok {
		return GoogleClaims{}, errors.New("Invalid Google JWT")
	}

	if claims.Issuer != "accounts.google.com" && claims.Issuer != "https://accounts.google.com" {
		return GoogleClaims{}, errors.New("iss is invalid")
	}

	// if claims.Audience != "YOUR_CLIENT_ID_HERE" {
	// 	return GoogleClaims{}, errors.New("aud is invalid")
	// }

	if claims.ExpiresAt < time.Now().UTC().Unix() {
		return GoogleClaims{}, errors.New("JWT is expired")
	}

	return *claims, nil
}
