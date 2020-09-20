# Online to ePub

This is a small program which allows me to download
my daily newspaper from <epaper.zeitungsverlag-aachen.de/>.

i have subscribed to the newspaper but was very unsatisfied
with the ePub version of it.

So I started to look into the online edition, which is a bit
better but not perfect yet.I already informed the newspaper
about the issues I found, like missing pictures every now and
then.

But I think the ePub my program produces is by far better
than the official one. So if you have subscribed to any of
the editions of Medienhaus Aachen, feel free to test this
converter.

## Requirements

You have to have a vaid subscription. Your credentials and
the edition of the newspaper have to be stored in environment
variables.

Example:

```shell
export AZAN_AUSGABE=az-d
export AZAN_USER=my.account@medienhaus.ac
export AZAN_PASS=MySecritPassword
```

### Available editions

To get the available editions simply call

```shell
azdl -?
```

No credentials required for this.

## Installation

You must have (obiously) go installed.

Download or clone the repository.

Change into the directory and enter `go build`.
This will build the binary `azdl` (short for Aachener
Zeitung DownLoad).

## Usage

The program `azdl` accepts up to two parameters:

1. The date of the release to download (or `latest`).
2. The edition to load.

The resulting epub will be stored in a file called

**edition**`-`**iso-date**`.epub`

Example:

`an-a1-2020-09-30.epub`

### `azdl`

This will download the latest release. It will use
the environment variables `AZAN_AUSGABE` to determine
which edition to load.

### `azdl` **edition**

This will download the latest release. It will use
the parameter to determine which edition to load.

### `azdl` **YYYYMMDD**

This will download the release of YYYY-MM-DD. The
edition to load will be determined by the
environment variable.

Instead of the date, `latest` can be used to load
the latest version.

### `azdl` **YYYYMMDD** **edition**

This will download the release of YYYY-MM-DD. The
edition is provided on the commandline.


