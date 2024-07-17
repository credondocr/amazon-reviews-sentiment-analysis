
# Sentiment Analysis of Amazon Reviews

This project performs sentiment analysis on Amazon product reviews to categorize them into positive and negative sentiments. It uses a pre-trained sentiment analysis model from the `github.com/cdipaolo/sentiment` package.

## Table of Contents

- [Overview](#overview)
- [Setup](#setup)
- [Usage](#usage)
- [Results](#results)
- [Contributing](#contributing)
- [License](#license)

## Overview

The goal of this project is to analyze Amazon product reviews and determine the overall sentiment (positive or negative) of the reviews. The sentiment analysis model processes each review and classifies it based on the text content. The project also calculates the average rating for negative reviews and the percentages of positive and negative reviews.

The reviews are read from a JSON Lines file, which can be downloaded from the provided URL.

## Setup

### Prerequisites

- Go 1.16+ installed on your machine. You can download it from [golang.org](https://golang.org/dl/).
- The `github.com/cdipaolo/sentiment` package for sentiment analysis.

### Installing Dependencies

First, clone the repository:

```sh
git clone https://github.com/credondocr/amazon-reviews-sentiment-analysis.git
cd amazon-reviews-sentiment-analysis
```

Then, install the necessary Go package:

```sh
go get github.com/cdipaolo/sentiment
```

## Usage

### Running the Program

1. Provide the URL of the review file to download and process using the `-url` flag. For example:

```sh
go run main.go -url https://datarepo.eng.ucsd.edu/mcauley_group/data/amazon_2023/raw/review_categories/All_Beauty.jsonl.gz
```

### Example URLs

You can use the following URLs to test the program:

- [Cell Phones and Accessories](https://datarepo.eng.ucsd.edu/mcauley_group/data/amazon_2023/raw/review_categories/Cell_Phones_and_Accessories.jsonl.gz)
- [All Beauty](https://datarepo.eng.ucsd.edu/mcauley_group/data/amazon_2023/raw/review_categories/All_Beauty.jsonl.gz)
- [Books](https://datarepo.eng.ucsd.edu/mcauley_group/data/amazon_2023/raw/review_categories/Books.jsonl.gz)

For more examples, visit [Amazon Reviews 2023](https://amazon-reviews-2023.github.io/).

### How It Works

1. **Model Restoration**: The sentiment analysis model is restored from the `sentiment` package.
2. **File Handling**: If the review file is not found, it will be downloaded and decompressed based on the provided URL.
3. **File Reading**: The program reads the reviews from the JSON Lines file.
4. **Processing Reviews**: Reviews are processed in chunks to optimize performance.
5. **Sentiment Analysis**: Each review is analyzed to determine if it is positive or negative.
6. **Results Calculation**:
   - Calculates the average rating for negative reviews.
   - Computes the total number and percentage of positive and negative reviews.

### Output

The program outputs:
- The average rating of negative reviews.
- The total number and percentage of positive and negative reviews.

## Results

Example output:

```
Average negative reviews: 3.45
Total negative reviews: 234
Total positive reviews: 765
Percentage negative reviews: 23.45%
Percentage positive reviews: 76.55%
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

### Steps to Contribute

1. Fork the repository.
2. Create a new branch (`git checkout -b feature/your-feature`).
3. Make your changes and commit them (`git commit -m 'Add some feature'`).
4. Push to the branch (`git push origin feature/your-feature`).
5. Open a pull request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
