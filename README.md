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

### Homebrew (Mac, Linux)

```shell
brew install robinovitch61/tap/webtoon-dl
```

To upgrade the version installed with homebrew:
```shell
brew update && brew upgrade webtoon-dl
```

### Download from Github (Mac, Linux, Windows)

Download the relevant binary for your operating system (MacOS = Darwin) from
the [latest github release](https://github.com/robinovitch61/webtoon-dl/releases). Unpack/extract it, then move the
binary or .exe to somewhere accessible in your `PATH`, e.g. `mv ./webtoon-dl /usr/local/bin`.

### Using [go installed on your machine](https://go.dev/doc/install) (Mac, Linux, Windows)

```shell
go install github.com/robinovitch61/webtoon-dl@latest
```

### Step by Step Windows Installation Instructions

1. Check the processor type you're using by going to System Information and looking at Processor. Go to [this page](https://github.com/robinovitch61/webtoon-dl/releases). If your Processor says "Intel" anywhere, download the file that ends in "Windows\_i386.tar.gz" by clicking it. If it says "Arm", instead click the file that ends in "Windows\_arm64.tar.gz".
2. Once the file is downloaded to your Downloads folder, you have to unzip and extract it. If you use [7zip](https://www.7-zip.org/), you can right click the file and hit "7zip -> Extract here", then right click the ".tar" file that's created and hit "7zip -> Extract here" again. You should end up with a file called "webtoon-dl.exe" in your Downloads folder.
3. If Windows thinks the .exe file is malware and deletes it automatically, you can prevent that by temporarily disabling "Real time protection" under "Virus & threat protection settings", following step 2 again to get back the .exe, then [adding an exclusion for it](https://support.microsoft.com/en-us/windows/add-an-exclusion-to-windows-security-811816c0-4dfd-af4a-47e4-c301afe13b26). This isn't malware, it's just some code I wrote in go, with all the source code available [here](https://github.com/robinovitch61/webtoon-dl/blob/main/main.go).
4. Once you have the .exe in your Downloads and Windows isn't going to auto-remove it, you can open the Command Prompt:
   1. First, type `cd Downloads` and hit enter to get to your Downloads folder
   2. Now confirm that "webtoon-dl.exe" shows up in the output when you type `dir` and hit enter
   3. Setup is over!
5. Still in the Command Prompt, now run, for example, `webtoon-dl.exe "https://www.webtoons.com/en/slice-of-life/bugtopia/ep-8-a-special-gift/viewer?title_no=4842&episode_no=8"`. This will run and print what it's doing, then output a PDF file of that comic in your Downloads folder.

## Build from Source (Mac, Linux, Windows)

Clone this repo, build from source with `cd <cloned_repo> && go build`. This will create the executable (e.g. `wander`) in the current directory.

