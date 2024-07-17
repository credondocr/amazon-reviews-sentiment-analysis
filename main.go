package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cdipaolo/sentiment"
	"github.com/schollz/progressbar/v3"
)

type Review struct {
	Rating           float64       `json:"rating"`
	Title            string        `json:"title"`
	Text             string        `json:"text"`
	Images           []interface{} `json:"images"`
	Asin             string        `json:"asin"`
	ParentAsin       string        `json:"parent_asin"`
	UserID           string        `json:"user_id"`
	Timestamp        int64         `json:"timestamp"`
	HelpfulVote      int           `json:"helpful_vote"`
	VerifiedPurchase bool          `json:"verified_purchase"`
}

type Counter struct {
	negative int32
	positive int32
	total    int32
}

func processLine(line string) (Review, error) {
	var review Review
	err := json.Unmarshal([]byte(line), &review)
	if err != nil {
		log.Printf("Error parsing line: %v, line: %s", err, line)
		return Review{}, err
	}
	return review, nil
}

func worker(ctx context.Context, model sentiment.Models, reviewChunks <-chan []Review, resultsCh chan<- float64, counter *Counter, wg *sync.WaitGroup, bar *progressbar.ProgressBar) {
	defer wg.Done()
	negatives := 0
	totalNegativeRating := 0.0
	for {
		select {
		case <-ctx.Done():
			return
		case chunk, ok := <-reviewChunks:
			if !ok {
				if negatives > 0 {
					resultsCh <- totalNegativeRating / float64(negatives)
				} else {
					resultsCh <- 0
				}
				return
			}
			for _, review := range chunk {
				analysis := model.SentimentAnalysis(review.Text, sentiment.English)
				atomic.AddInt32(&counter.total, 1)
				if analysis.Score == 0 {
					totalNegativeRating += review.Rating
					negatives++
					atomic.AddInt32(&counter.negative, 1)
				} else {
					atomic.AddInt32(&counter.positive, 1)
				}
				bar.Add(1)
			}
		}
	}
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return err
	}

	bar := progressbar.DefaultBytes(
		int64(size),
		"Downloading",
	)

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
	return err
}

func decompressGzip(src, dest string) error {
	gzFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer gzFile.Close()

	gzReader, err := gzip.NewReader(gzFile)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	outFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, gzReader)
	return err
}

func countLines(filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

func main() {
	urlFlag := flag.String("url", "", "URL of the review file to download")
	flag.Parse()

	if *urlFlag == "" {
		log.Fatalf("URL is required. Use -url to specify the URL of the review file.")
	}

	parsedURL, err := url.Parse(*urlFlag)
	if err != nil {
		log.Fatalf("Invalid URL: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	model, err := sentiment.Restore()
	if err != nil {
		log.Fatalf("Error loading sentiment model: %v", err)
	}

	dataGzFileName := filepath.Base(parsedURL.Path)
	dataFileName := strings.TrimSuffix(dataGzFileName, ".gz")

	// Check if the file exists, if not, download and decompress it
	if _, err := os.Stat(dataFileName); os.IsNotExist(err) {
		fmt.Printf("File %s not found. Downloading...\n", dataFileName)
		err := downloadFile(*urlFlag, dataGzFileName)
		if err != nil {
			log.Fatalf("Error downloading file: %v", err)
		}

		fmt.Printf("Decompressing %s...\n", dataGzFileName)
		err = decompressGzip(dataGzFileName, dataFileName)
		if err != nil {
			log.Fatalf("Error decompressing file: %v", err)
		}
		fmt.Printf("File downloaded and decompressed successfully.\n")
	}

	// Count the number of lines for the progress bar
	totalLines, err := countLines(dataFileName)
	if err != nil {
		log.Fatalf("Error counting lines in file: %v", err)
	}

	file, err := os.Open(dataFileName)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	reviewChunks := make(chan []Review, 100)
	resultsCh := make(chan float64)
	var wg sync.WaitGroup
	var counter Counter
	numWorkers := 10
	chunkSize := 100

	// Progress bar for reading and processing reviews
	bar := progressbar.NewOptions(totalLines,
		progressbar.OptionSetDescription("Processing reviews"),
		progressbar.OptionSetWidth(40),
		progressbar.OptionShowCount(),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionUseANSICodes(true),
	)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(ctx, model, reviewChunks, resultsCh, &counter, &wg, bar)
	}

	go func() {
		defer close(reviewChunks)
		var chunk []Review
		for scanner.Scan() {
			line := scanner.Text()
			review, err := processLine(line)
			if err != nil {
				log.Println("Error unmarshalling line:", err)
				continue
			}
			chunk = append(chunk, review)
			if len(chunk) == chunkSize {
				select {
				case reviewChunks <- chunk:
					chunk = nil
				case <-ctx.Done():
					return
				}
			}
		}
		if len(chunk) > 0 {
			reviewChunks <- chunk
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}()

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	totalNegativeRating := 0.0
	negativeReviewCount := 0
	for result := range resultsCh {
		if result > 0 {
			totalNegativeRating += result
			negativeReviewCount++
		}
	}

	if negativeReviewCount > 0 {
		averageNegativeRating := totalNegativeRating / float64(negativeReviewCount)
		fmt.Printf("Average negative reviews: %.2f\n", averageNegativeRating)
	}

	negativePercentage := float64(counter.negative) / float64(counter.total) * 100
	positivePercentage := float64(counter.positive) / float64(counter.total) * 100

	fmt.Printf("Total negative reviews: %d\n", counter.negative)
	fmt.Printf("Total positive reviews: %d\n", counter.positive)
	fmt.Printf("Percentage negative reviews: %.2f%%\n", negativePercentage)
	fmt.Printf("Percentage positive reviews: %.2f%%\n", positivePercentage)
}
