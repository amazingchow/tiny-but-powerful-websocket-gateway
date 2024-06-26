package jwt

import (
	"context"
	"crypto/rsa"
	"errors"
	"os"
	"testing/fstest"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"

	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/common/logger"
	"github.com/amazingchow/tiny-but-powerful-websocket-gateway/internal/service/common"
)

var (
	ErrUnexpectedTokenSignatureMethod = errors.New("unexpected token signature method, we only accept RS256-signed token")
	ErrInvalidTokenClaims             = errors.New("invalid token claims")
)

// JWTManager is a JSON Web Token manager, JWT.io has a great introduction(https://jwt.io/introduction) to JSON Web Tokens.
var jwtMgr *JWTManager

type JWTManager struct {
	logger *logrus.Entry

	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

func SetupJWTManager(publicKeyFile, privateKeyFile string, fs fstest.MapFS) {
	var pemData []byte
	var err error
	if fs != nil {
		pemData, err = fs.ReadFile(publicKeyFile)
		if err != nil {
			logger.GetGlobalLogger().WithError(err).Fatal("Failed to read public key file.")
		}
	} else {
		pemData, err = os.ReadFile(publicKeyFile)
		if err != nil {
			logger.GetGlobalLogger().WithError(err).Fatal("Failed to read public key file.")
		}
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(pemData)
	if err != nil {
		logger.GetGlobalLogger().WithError(err).Fatal("Failed to parse public key.")
	}
	if fs != nil {
		pemData, err = fs.ReadFile(privateKeyFile)
		if err != nil {
			logger.GetGlobalLogger().WithError(err).Fatal("Failed to read private key file.")
		}
	} else {
		pemData, err = os.ReadFile(privateKeyFile)
		if err != nil {
			logger.GetGlobalLogger().WithError(err).Fatal("Failed to read private key file.")
		}
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(pemData)
	if err != nil {
		logger.GetGlobalLogger().WithError(err).Fatal("Failed to parse private key.")
	}

	jwtMgr = &JWTManager{
		logger:     logger.GetGlobalLogger().WithField("infra", "jwt"),
		publicKey:  publicKey,
		privateKey: privateKey,
	}
}

func GetJWTManager() *JWTManager {
	return jwtMgr
}

type UserClaims struct {
	jwt.RegisteredClaims
	Account string `json:"account"`
}

func (mgr *JWTManager) BuildAndSignToken(ctx context.Context, account string) (tokenString string, err error) {
	_logger := mgr.logger.
		WithField(common.LoggerKeyTraceId, ctx.Value(common.ContextKeyTraceId).(string)).
		WithField(common.LoggerKeySpanId, ctx.Value(common.ContextKeySpanId).(string))

	// 创建一个新的token
	token := jwt.New(jwt.SigningMethodRS256)
	// 设置claims
	claims := &UserClaims{
		Account: account,
	}
	token.Claims = claims
	// 使用RSA私钥对token进行签名
	tokenString, err = token.SignedString(mgr.privateKey)
	if err != nil {
		_logger.WithError(err).Error("Failed to sign jwt token.")
		return
	}
	return
}

func (mgr *JWTManager) ParseAndValidateToken(ctx context.Context, account string, tokenString string) (valid bool, err error) {
	_logger := mgr.logger.
		WithField(common.LoggerKeyTraceId, ctx.Value(common.ContextKeyTraceId).(string)).
		WithField(common.LoggerKeySpanId, ctx.Value(common.ContextKeySpanId).(string))

	// 解析token
	token, err := jwt.ParseWithClaims(
		tokenString,
		&UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			_, ok := token.Method.(*jwt.SigningMethodRSA)
			if !ok {
				return nil, ErrUnexpectedTokenSignatureMethod
			}
			return mgr.publicKey, nil
		},
	)
	if err != nil {
		_logger.WithError(err).Error("Failed to parse jwt token.")
		return
	}
	// 验证token
	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		err = ErrInvalidTokenClaims
		_logger.WithError(err).Error("Failed to parse jwt token.")
		return
	}
	// 验证claims
	valid = claims.Account == account
	return
}
