# jupyteach-cli

Command line interface for managing Jupyteach course content.

## Installation

Grab a binary from the [releases](https://github.com/sglyon/jupyteach-cli/releases) page

After downloading, log in to your Jupyteach instance, go to your user settings page, and generate an API token. Copy this to your clipboard

Then run `jupyteach login` and provide the API token when prompted

## Usage

### Starting a new course

1. Create the course on the Jupyteach website
2. Grab the course slug from the URL (Something like `.../course/<slug>/...`)
3. Run `jupyteach clone <slug>`

### Clone an existing course

1. Visit the course page on your Jupyteach instance
2. Grab the course slug from the URL (Something like `.../course/<slug>/...`)
3. Run `jupyteach clone <slug>`

### Changing course content

1. Make changes to your local course content
2. Commit the changes using your standard `git` workflow
3. Run `jupyteach push` to push the changes to the server

### Pulling remote changes

1. Run `jupyteach pull` to pull the latest changes from the server
