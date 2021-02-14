package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"reddit-image-recognition-fetcher/shared/environment"
	"reddit-image-recognition-fetcher/shared/s3"
	urlValidator "reddit-image-recognition-fetcher/shared/url-validator"
	"strings"
	"sync"

	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/vartanbeno/go-reddit/v2/reddit"
)

var (
	redditImageDomains = []string{
		"https://i.redd.it/",
		"https://i.imgur.com/",
	}
	redditImagesBucket = environment.GetString("REDDIT_IMAGES_BUCKET", "ra-reddit-images")
	subreddit          = environment.GetString("SUBREDDIT", "Konosuba")
)

const (
	maxPosts   = 100
	iterations = 2
)

// RedditRequest is the struct for the lambda request data
type RedditRequest struct {
	*events.APIGatewayProxyRequest
}

func apiGatewayHandler(ctx context.Context, request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	req := RedditRequest{
		APIGatewayProxyRequest: &request,
	}

	err := req.fetchImagesUrls(ctx)
	if err != nil {
		data, _ := json.Marshal(map[string]string{
			"error": err.Error(),
		})

		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       string(data),
		}, nil
	}

	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
	}, nil
}

func (req RedditRequest) fetchImagesUrls(ctx context.Context) error {
	after := ""

	var wg sync.WaitGroup

	for i := 0; i < iterations; i++ {
		posts, resp, err := reddit.DefaultClient().Subreddit.TopPosts(ctx, subreddit, &reddit.ListPostOptions{
			ListOptions: reddit.ListOptions{
				Limit: maxPosts,
				After: after,
			},
			Time: "day",
		})
		if err != nil {
			return err
		}

		for _, post := range posts {
			wg.Add(1)
			go req.uploadToS3(ctx, post.URL, post.ID, &wg)
		}

		// Calling again with fewer results than available will return the same results
		if len(posts) < maxPosts {
			break
		}

		after = resp.After
	}

	wg.Wait()

	return nil
}

func (req RedditRequest) uploadToS3(ctx context.Context, url string, name string, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()

	if !isValidRedditImageURL(url) {
		return
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	extension := strings.Split(resp.Header.Get("Content-Type"), "/")[1]

	err = s3.UploadObject(ctx, redditImagesBucket, name+"."+extension, body)
	if err != nil {
		fmt.Println(err)
	}
}

func isValidRedditImageURL(rawURL string) bool {
	for _, domain := range redditImageDomains {
		if urlValidator.IsValidURLOfGivenDomain(rawURL, domain) {
			return true
		}
	}

	return false
}

func main() {
	lambda.Start(apiGatewayHandler)
}
