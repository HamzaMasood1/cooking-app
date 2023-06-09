package handlers

import (
	"HamzaMasood1/cooking-app/api/models"
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/rs/xid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	collection *mongo.Collection
	ctx        context.Context
}

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

type JWTOutput struct {
	Token   string    `json:"token"`
	Expires time.Time `json:"expires"`
}

func NewAuthHandler(ctx context.Context, collection *mongo.Collection) *AuthHandler {
	return &AuthHandler{
		collection: collection,
		ctx:        ctx,
	}
}

// 1. Encode request body into a user struct and verify credentials are correct
// 2. Encode a jwt token and with expiration of 10 minutes
// 3. jwt signature = (header in base64)+(payload in base64)+(secret key)
func (handler *AuthHandler) SignInHandler(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cur := handler.collection.FindOne(handler.ctx, bson.M{
		"username": user.Username,
	})
	//incorrect username
	if cur.Err() != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}
	var userdb models.User
	cur.Decode(&userdb)
	//incorrect password
	if !Verify(userdb.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}
	expirationTime := time.Now().Add(10 * time.Minute)
	claims := &Claims{
		Username: user.Username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	jwtOutput := JWTOutput{
		Token:   tokenString,
		Expires: expirationTime,
	}

	sessionToken := xid.New().String()
	session := sessions.Default(c)
	session.Set("username", user.Username)
	session.Set("token", sessionToken)
	session.Save()

	c.JSON(http.StatusOK, gin.H{"cookie-session": "created", "jwtOuutput": jwtOutput})
}
func Verify(hashed, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
	return err == nil
}

func (handler *AuthHandler) SignOutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.JSON(http.StatusOK, gin.H{"message": "Signed out..."})
}

func (handler *AuthHandler) RefreshHandler(c *gin.Context) {
	tokenValue := c.GetHeader("Authorization")
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenValue, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		// if err.(*jwt.ValidationError).Errors != jwt.ValidationErrorExpired {
		// 	c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		// 	return
		// }
		validationErr, ok := err.(*jwt.ValidationError)
		if !ok || validationErr.Errors != jwt.ValidationErrorExpired {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
	}

	if time.Unix(claims.ExpiresAt, 0).Sub(time.Now()) > 30*time.Second {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is not expired yet"})
		return
	}

	expirationTime := time.Now().Add(5 * time.Minute)
	claims.ExpiresAt = expirationTime.Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	jwtOutput := JWTOutput{
		Token:   tokenString,
		Expires: expirationTime,
	}
	c.JSON(http.StatusOK, jwtOutput)
}

// CustomClaims contains custom data we want from the token.
type CustomClaims struct {
	Scope string `json:"scope"`
}

// Validate does nothing for this example, but we need
// it to satisfy validator.CustomClaims interface.
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

func (handler *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		////////////auth with api token
		// if c.GetHeader("X_API_KEY") != os.Getenv("X_API_KEY") {
		// 	c.AbortWithStatusJSON(401, gin.H{"error": "API key not provided or invalid"})
		// }
		// c.Next()
		////////////
		tokenValue := c.GetHeader("Authorization")
		if tokenValue == "" {
			session := sessions.Default(c)
			sessionToken := session.Get("token")
			if sessionToken == nil {
				c.JSON(http.StatusForbidden, gin.H{"message": "not logged"})
				c.Abort()
			}
			c.Next()
		} else {
			///////////////local jwt validation
			// claims := &Claims{}
			// tkn, err := jwt.ParseWithClaims(tokenValue, claims,
			// 	func(token *jwt.Token) (interface{}, error) {
			// 		return []byte(os.Getenv("JWT_SECRET")), nil
			// 	})
			// if err != nil || tkn == nil || !tkn.Valid {
			// 	c.AbortWithStatusJSON(401, gin.H{"error": "API key not provided or invalid"})
			// }
			// c.Next()
			// Set up the validator.

			//////////////////auth0 validation
			issuerURL, _ := url.Parse("https://" + os.Getenv("AUTH0_DOMAIN") + "/")
			provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)
			jwtValidator, err := validator.New(
				provider.KeyFunc,
				validator.RS256,
				issuerURL.String(),
				[]string{os.Getenv("AUTH0_AUDIENCE")},
				validator.WithCustomClaims(
					func() validator.CustomClaims {
						return &CustomClaims{}
					},
				),
				validator.WithAllowedClockSkew(30*time.Second),
			)
			if err != nil {
				log.Fatalf("failed to set up the validator: %v", err)
			}

			errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
				log.Printf("Encountered error while validating JWT: %v", err)
			}

			middleware := jwtmiddleware.New(
				jwtValidator.ValidateToken,
				jwtmiddleware.WithErrorHandler(errorHandler),
			)
			encounteredError := true
			var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
				encounteredError = false
				c.Request = r
				c.Next()
			}

			middleware.CheckJWT(handler).ServeHTTP(c.Writer, c.Request)

			if encounteredError {
				c.AbortWithStatusJSON(
					http.StatusUnauthorized,
					map[string]string{"message": "JWT is invalid."},
				)
			}
		}

	}
}
