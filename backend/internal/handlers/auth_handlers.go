package handlers

import (
	"context"
	"encoding/json"
	"family-potluck/backend/internal/models"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/api/idtoken"
)

var jwtKey = []byte(os.Getenv("JWT_SECRET"))

func init() {
	if len(jwtKey) == 0 {
		jwtKey = []byte("my_secret_key") // Fallback for dev
	}
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateToken(userID string) (string, error) {
	expirationTime := time.Now().Add(72 * time.Hour)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func setTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Now().Add(72 * time.Hour),
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		// Secure:   true, // Uncomment in production with HTTPS
	})
}

func (s *Server) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	// Ideally, validate the audience (your Client ID)
	payload, err := idtoken.Validate(ctx, req.IDToken, os.Getenv("GOOGLE_CLIENT_ID"))
	if err != nil {
		http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
		return
	}

	email := payload.Claims["email"].(string)
	name := payload.Claims["name"].(string)
	picture := payload.Claims["picture"].(string)
	sub := payload.Subject // Google ID

	collection := s.DB.GetCollection("families")
	var family models.Family
	err = collection.FindOne(ctx, bson.M{"email": email}).Decode(&family)

	if err == mongo.ErrNoDocuments {
		// Create new user
		family = models.Family{
			ID:       primitive.NewObjectID(),
			Name:     name,
			Email:    email,
			GoogleID: sub,
			Picture:  picture,
		}
		_, err = collection.InsertOne(ctx, family)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		// Update existing user info if needed
		update := bson.M{
			"$set": bson.M{
				"name":      name,
				"picture":   picture,
				"google_id": sub,
			},
		}
		_, err = collection.UpdateOne(ctx, bson.M{"_id": family.ID}, update)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	token, err := GenerateToken(family.ID.Hex())
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	setTokenCookie(w, token)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(family)
}

func (s *Server) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Logged out"))
}

func (s *Server) GetMe(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("token")
	if err != nil {
		if err == http.ErrNoCookie {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	tokenStr := c.Value
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		http.Error(w, "Invalid token data", http.StatusUnauthorized)
		return
	}

	collection := s.DB.GetCollection("families")
	var family models.Family
	err = collection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&family)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(family)
}
