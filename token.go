package main

import (
	"encoding/json"
	"fmt"
	"github.com/sandertv/gophertunnel/minecraft/auth"
	"golang.org/x/oauth2"
	"os"
)

var cachePath = "./token_cache.json"

func GetAccountToken() (*oauth2.TokenSource, error) {
	data, err := os.ReadFile(cachePath)
	if err == nil {
		var token oauth2.Token
		err = json.Unmarshal(data, &token)
		if err == nil {
			tkn := auth.RefreshTokenSource(&token)
			return &tkn, nil
		}
	}
	token, err := auth.RequestLiveToken()
	if err != nil {
		return nil, fmt.Errorf("error getting live token: %w", err)
	}
	data, err = json.Marshal(token)
	if err != nil {
		return nil, fmt.Errorf("error marshaling token: %w", err)
	}
	err = os.WriteFile(cachePath, data, 0600)
	if err != nil {
		return nil, fmt.Errorf("error writing token cache: %w", err)
	}
	src := auth.RefreshTokenSource(token)
	return &src, nil
}
