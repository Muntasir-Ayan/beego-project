package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"bytes"
	// "time"
	beego "github.com/beego/beego/v2/server/web"
)

type Vote struct {
	ImageID string `json:"image_id"`
	Value   int    `json:"value"`
}

type response struct {
	data []byte
	err  error
}

type VoteResponse struct {
    ID          int       `json:"id"`
    ImageID     string    `json:"image_id"`
    Value       int       `json:"value"`
    SubID       *string   `json:"sub_id,omitempty"` // Optional
    CreatedAt   string    `json:"created_at"`      // Assuming this field is available
    CountryCode string    `json:"country_code"`    // You can add this if required
    Image       Image     `json:"image"`           // Nested Image struct
}

type CatImage struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Breed struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Origin      string `json:"origin"`
	ImageID     string `json:"reference_image_id"`
}

type BreedImage struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Breeds []struct {
		Name        string `json:"name"`
		Origin      string `json:"origin"`
		Description string `json:"description"`
		Wikipedia   string `json:"wikipedia_url"`
	} `json:"breeds"`
}

type Image struct {
    ID  string `json:"id"`
    URL string `json:"url"`
}

type Favourite struct {
	ImageID string `json:"image_id"`
	SubID   string `json:"sub_id,omitempty"` // Optional user ID
}

type FavouriteResponse struct {
	ID        int      `json:"id"`
	UserID    string   `json:"user_id"`
	ImageID   string   `json:"image_id"`
	SubID     *string  `json:"sub_id,omitempty"`
	CreatedAt string   `json:"created_at"`
	Image     Image    `json:"image"` // Nested Image struct
}

type CustomController struct {
	beego.Controller
}

// Fetches a random cat image
func (c *CustomController) Get() {
	apiKey, _ := beego.AppConfig.String("catapi_key")
	url := "https://api.thecatapi.com/v1/images/search"

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.handleError(err)
		return
	}
	req.Header.Set("x-api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		c.handleError(err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.handleError(err)
		return
	}

	var catImages []CatImage
	if err := json.Unmarshal(body, &catImages); err != nil || len(catImages) == 0 {
		c.handleError(err)
		return
	}

	c.Data["CatImageURL"] = catImages[0].URL
	c.TplName = "custom_page.tpl"
}

// Fetches list of breeds
func (c *CustomController) GetBreeds() {
	apiKey, _ := beego.AppConfig.String("catapi_key")
	url := "https://api.thecatapi.com/v1/breeds"

	// Create a channel for the API response
	respChannel := make(chan response)

	// Use a Goroutine to fetch breeds
	go func() {
		client := &http.Client{}
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("x-api-key", apiKey)

		resp, err := client.Do(req)
		if err != nil {
			respChannel <- response{nil, err}
			return
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		respChannel <- response{body, nil}
	}()

	// Wait for the response from the Goroutine
	resp := <-respChannel

	if resp.err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to fetch breeds")
		return
	}

	var breeds []Breed
	if err := json.Unmarshal(resp.data, &breeds); err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to parse breeds")
		return
	}

	c.Data["json"] = breeds
	c.ServeJSON()
}

// Fetches images and details for a specific breed
func (c *CustomController) GetBreedImages() {
	apiKey, _ := beego.AppConfig.String("catapi_key")
	breedID := c.GetString("breed_id")
	if breedID == "" {
		c.CustomAbort(http.StatusBadRequest, "Missing breed ID")
		return
	}

	url := fmt.Sprintf("https://api.thecatapi.com/v1/images/search?limit=8&breed_ids=%s", breedID)

	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("x-api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to fetch breed images")
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var images []BreedImage
	if err := json.Unmarshal(body, &images); err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to parse breed images")
		return
	}

	// Log breed information for debugging
	for _, image := range images {
		for _, breed := range image.Breeds {
			fmt.Printf("Breed Name: %s, Wikipedia: %s\n", breed.Name, breed.Wikipedia)
		}
	}

	c.Data["json"] = images
	c.ServeJSON()
}

func (c *CustomController) CreateVote() {
	apiKey, _ := beego.AppConfig.String("catapi_key")
	imageID := c.GetString("image_id")
	voteValue, err := c.GetInt("value")
	if err != nil || imageID == "" {
		c.CustomAbort(http.StatusBadRequest, "Invalid vote data")
		return
	}

	url := "https://api.thecatapi.com/v1/votes"

	vote := Vote{
		ImageID: imageID,
		Value:   voteValue,
	}

	voteData, _ := json.Marshal(vote)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(voteData))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to create vote")
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	c.Data["json"] = string(body) // Return response from the API
	c.ServeJSON()
}

func (c *CustomController) GetVotes() {
	apiKey, _ := beego.AppConfig.String("catapi_key")
	url := "https://api.thecatapi.com/v1/votes"
	limit := c.GetString("limit") // No default value here
	order := c.GetString("order", "DESC") // Default to DESC if not provided

	// Construct query URL
	query := url + "?order=" + order
	if limit != "" {
		query += "&limit=" + limit
	}

	// Create a channel for the API response
	respChannel := make(chan response)

	// Use a Goroutine to fetch votes
	go func() {
		req, _ := http.NewRequest("GET", query, nil)
		req.Header.Set("x-api-key", apiKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			respChannel <- response{nil, err}
			return
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		respChannel <- response{body, nil}
	}()

	// Wait for the response from the Goroutine
	resp := <-respChannel

	if resp.err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to fetch votes")
		return
	}

	var votes []VoteResponse
	if err := json.Unmarshal(resp.data, &votes); err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to parse votes")
		return
	}

	var formattedVotes []map[string]interface{}
	for _, vote := range votes {
		imageURL := fmt.Sprintf("https://cdn2.thecatapi.com/images/%s", vote.ImageID)

		formattedVote := map[string]interface{}{
			"id":          vote.ID,
			"image_id":    vote.ImageID,
			"sub_id":      vote.SubID,
			"created_at":  vote.CreatedAt,
			"value":       vote.Value,
			"country_code": "JP",
			"image": map[string]interface{}{
				"id":  vote.ImageID,
				"url": imageURL,
			},
		}

		formattedVotes = append(formattedVotes, formattedVote)
	}

	c.Data["json"] = formattedVotes
	c.ServeJSON()
}


// CreateFavourite: Handle the creation of a favourite
func (c *CustomController) CreateFavourite() {
	apiKey, _ := beego.AppConfig.String("catapi_key")
	imageID := c.GetString("image_id")
	subID := c.GetString("sub_id") // Optional user ID

	if imageID == "" {
		c.CustomAbort(http.StatusBadRequest, "Image ID is required")
		return
	}

	url := "https://api.thecatapi.com/v1/favourites"
	fav := Favourite{
		ImageID: imageID,
		SubID:   subID,
	}

	favData, _ := json.Marshal(fav)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(favData))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to create favourite")
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	var favResponse FavouriteResponse
	if err := json.Unmarshal(body, &favResponse); err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to parse response")
		return
	}

	c.Data["json"] = favResponse
	c.ServeJSON()
}

// GetFavourites: Fetch all favourites for the user
func (c *CustomController) GetFavourites() {
	apiKey, _ := beego.AppConfig.String("catapi_key")
	url := "https://api.thecatapi.com/v1/favourites"

	// Fetch favourites
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("x-api-key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to fetch favourites")
		return
	}
	defer resp.Body.Close()

	// Check the status code
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("Error: %s\nResponse: %s\n", resp.Status, string(body))
		c.CustomAbort(http.StatusInternalServerError, "Failed to fetch favourites")
		return
	}

	// Read and unmarshal the response body
	body, _ := ioutil.ReadAll(resp.Body)
	var favourites []FavouriteResponse
	if err := json.Unmarshal(body, &favourites); err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to parse favourites")
		return
	}

	// Add image URL to each favourite
	var formattedFavourites []map[string]interface{}

	for _, fav := range favourites {
		// Fetch the image details for the current favourite
		imageURL := fmt.Sprintf("https://cdn2.thecatapi.com/images/%s", fav.ImageID)
		
		// Create the formatted favourite response
		formattedFavourite := map[string]interface{}{
			"id":          fav.ID,
			"image_id":    fav.ImageID,
			"sub_id":      fav.SubID, // May be null
			"created_at":  fav.CreatedAt,
			"image": map[string]interface{}{
				"id":  fav.ImageID,
				"url": imageURL, // Use full image URL
			},
		}

		// Append to the formatted favourites array
		formattedFavourites = append(formattedFavourites, formattedFavourite)
	}

	// Send the formatted response
	c.Data["json"] = formattedFavourites
	c.ServeJSON()
}


func (c *CustomController) DeleteFavourite() {
	apiKey, _ := beego.AppConfig.String("catapi_key")
	favouriteID := c.Ctx.Input.Param(":id") // Get the `id` from the URL

	if favouriteID == "" {
		c.CustomAbort(http.StatusBadRequest, "Favourite ID is required")
		return
	}

	url := fmt.Sprintf("https://api.thecatapi.com/v1/favourites/%s", favouriteID)

	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Set("x-api-key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to delete favourite")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("Error: %s\nResponse: %s\n", resp.Status, string(body))
		c.CustomAbort(http.StatusInternalServerError, "Failed to delete favourite")
		return
	}

	c.Data["json"] = map[string]string{"message": "Favourite deleted successfully"}
	c.ServeJSON()
}



// Error handling for XMLHttpRequest
func (c *CustomController) handleError(err error) {
	if c.Ctx.Input.Header("X-Requested-With") == "XMLHttpRequest" {
		c.CustomAbort(http.StatusInternalServerError, err.Error())
	} else {
		c.Data["CatImageURL"] = ""
		c.TplName = "custom_page.tpl"
	}
}
