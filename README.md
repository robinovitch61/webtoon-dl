# webtoon-dl

Download [webtoon](https://www.webtoons.com/en/) comics as PDFs using a terminal/command line.

## Usage

```shell
# download single episodes
webtoon-dl "<your-webtoon-episode-url>"

# download entire series, default 10 episodes per pdf
webtoon-dl "<your-webtoon-series-url>"

# specify a range of episodes (inclusive on both ends)
webtoon-dl --min-ep=10 --max-ep=20 "<your-webtoon-series-url>"

# change the number of episodes per file, e.g. this would create 11 files
webtoon-dl --min-ep=10 --max-ep=20 --eps-per-file=1 "<your-webtoon-series-url>"

# download entire series into a single file (GENERALLY NOT RECOMMENDED)
webtoon-dl --eps-per-file=1000000 "<your-webtoon-series-url>"
```

> [!IMPORTANT]
> The episode numbers specified in `--min-ep` and `--max-ep` will correspond to the URL parameter `&episode_no=`, which may be different from the episode number in the title

> [!IMPORTANT]
> Some terminal settings (e.g. [Oh My Zsh](https://ohmyz.sh)) make it so pasted URLs will be [automatically escaped](https://github.com/ohmyzsh/ohmyzsh/issues/7632).
> You want to EITHER surround your unescaped webtoon URL with double quotes (otherwise you'll get something like a "no matches found" error) OR leave the double quotes off escaped URLs.
> So either of these will work:
> - `webtoon-dl "https://www.webtoons.com/.../list?title_no=123"`
> - `webtoon-dl https://www.webtoons.com/.../list\?title_no\=123`
>
> But this won't work:
> - `webtoon-dl "https://www.webtoons.com/.../list\?title_no\=123"`

## Installation

```shell
# homebrew
brew install robinovitch61/tap/webtoon-dl

# upgrade using homebrew
brew update && brew upgrade webtoon-dl

# windows with winget
winget install robinovitch61.webtoon-dl

# windows with scoop
scoop bucket add robinovitch61 https://github.com/robinovitch61/scoop-bucket
scoop install webtoon-dl

# windows with chocolatey
choco install webtoon-dl

# with go (https://go.dev/doc/install)
go install github.com/robinovitch61/webtoon-dl@latest
```

Alternatively, download the relevant binary for your operating system (MacOS = Darwin) from
the [latest github release](https://github.com/robinovitch61/webtoon-dl/releases). Unpack/extract it, then move the
binary or .exe to somewhere accessible in your `PATH`, e.g. `mv ./webtoon-dl /usr/local/bin`.

## Build from Source (Mac, Linux, Windows)

Clone this repo, build from source with `cd <cloned_repo> && go build`. This will create the executable (e.g. `webtoon-dl`) in the current directory.

