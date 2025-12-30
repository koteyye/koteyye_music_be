package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// SwaggerExamples - примеры для сложных типов ответов
var SwaggerExamples = map[string]interface{}{
	"models.TrackListResponse": map[string]interface{}{
		"tracks": []map[string]interface{}{
			{
				"id":               "550e8400-e29b-41d4-a716-446655440000",
				"user_id":          1,
				"title":            "Bohemian Rhapsody",
				"artist":           "Queen",
				"album":            "A Night at the Opera",
				"duration_seconds": 354,
				"s3_audio_key":     "audio/550e8400-e29b-41d4-a716-446655440000.mp3",
				"s3_image_key":     "images/550e8400-e29b-41d4-a716-446655440000.jpg",
				"plays_count":      1250,
				"likes_count":      87,
				"is_liked":         true,
				"created_at":       "2024-01-15T10:30:00Z",
			},
			{
				"id":               "660e9511-f30c-52e5-b827-557766551111",
				"user_id":          2,
				"title":            "Stairway to Heaven",
				"artist":           "Led Zeppelin",
				"album":            "Led Zeppelin IV",
				"duration_seconds": 482,
				"s3_audio_key":     "audio/660e9511-f30c-52e5-b827-557766551111.mp3",
				"s3_image_key":     "images/660e9511-f30c-52e5-b827-557766551111.jpg",
				"plays_count":      2100,
				"likes_count":      156,
				"is_liked":         false,
				"created_at":       "2024-01-16T14:20:00Z",
			},
		},
		"pagination": map[string]interface{}{
			"page":  1,
			"limit": 20,
			"total": 42,
		},
	},
	"models.UserTracksResponse": map[string]interface{}{
		"tracks": []map[string]interface{}{
			{
				"id":               "550e8400-e29b-41d4-a716-446655440000",
				"user_id":          1,
				"title":            "My Song",
				"artist":           "Me",
				"album":            "My Album",
				"duration_seconds": 180,
				"s3_audio_key":     "audio/550e8400-e29b-41d4-a716-446655440000.mp3",
				"s3_image_key":     "images/550e8400-e29b-41d4-a716-446655440000.jpg",
				"plays_count":      45,
				"likes_count":      3,
				"is_liked":         false,
				"created_at":       "2024-01-15T10:30:00Z",
			},
		},
	},
	"models.ToggleLikeResponse": map[string]interface{}{
		"liked":       true,
		"likes_count": 88,
	},
	"models.AuthResponse": map[string]interface{}{
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxIiwiZXhwIjoxNzA1MzQzODAwfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
		"user": map[string]interface{}{
			"id":          1,
			"email":       "user@example.com",
			"provider":    "local",
			"external_id": "",
			"role":        "user",
			"created_at":  "2024-01-15T10:30:00Z",
		},
	},
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run add_swagger_examples.go <path-to-swagger.json>")
		os.Exit(1)
	}

	swaggerPath := os.Args[1]

	// Читаем swagger.json
	data, err := ioutil.ReadFile(swaggerPath)
	if err != nil {
		fmt.Printf("Error reading swagger.json: %v\n", err)
		os.Exit(1)
	}

	var swagger map[string]interface{}
	if err := json.Unmarshal(data, &swagger); err != nil {
		fmt.Printf("Error parsing swagger.json: %v\n", err)
		os.Exit(1)
	}

	// Получаем definitions
	definitions, ok := swagger["definitions"].(map[string]interface{})
	if !ok {
		fmt.Println("No definitions found in swagger.json")
		os.Exit(1)
	}

	// Добавляем примеры к каждой модели
	for modelName, example := range SwaggerExamples {
		def, ok := definitions[modelName].(map[string]interface{})
		if !ok {
			continue
		}

		// Добавляем пример к определению
		def["example"] = example
		definitions[modelName] = def
	}

	// Записываем обратно в swagger.json
	swagger["definitions"] = definitions

	output, err := json.MarshalIndent(swagger, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling swagger.json: %v\n", err)
		os.Exit(1)
	}

	if err := ioutil.WriteFile(swaggerPath, output, 0644); err != nil {
		fmt.Printf("Error writing swagger.json: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Examples added to %s\n", swaggerPath)
}
