# webtoon-dl

Download [webtoon](https://www.webtoons.com/en/) comics as PDFs using a terminal/command line.

## Usage

```shell
# download single episodes
webtoon-dl <your-webtoon-episode-url>

# download entire series
webtoon-dl <your-webtoon-series-url>

# create single pdfs from a range of episodes (inclusive)
webtoon-dl --min-ep=10 --max-ep=20 <your-webtoon-series-url>
```

## Installation

### Homebrew

```shell
brew install robinovitch61/tap/webtoon-dl

# to upgrade
brew update && brew upgrade webtoon-dl
```

### Download from Github

Download the relevant binary for your operating system (MacOS = Darwin) from
the [latest github release](https://github.com/robinovitch61/webtoon-dl/releases). Unpack/extract it, then move the
binary or .exe to somewhere accessible in your `path`, e.g. `mv ./webtoon-dl /usr/local/bin`.

### Using [go installed on your machine](https://go.dev/doc/install)

```shell
go install github.com/robinovitch61/webtoon-dl@latest
```

### Build from Source

Clone this repo, build from source with `cd <cloned_repo> && go build`, then move the binary to somewhere accessible in
your `path`, e.g. `mv ./webtoon-dl /usr/local/bin`.
