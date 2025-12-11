# Docker Setup and GitHub Packages Release

## Docker Build

Build the Docker image:

```bash
docker build -t xcstrings-translator .
```

## Running the Container

Run the container with your xcstrings files:

```bash
# Mount a directory containing your xcstrings files
docker run -v /path/to/your/files:/workspace -w /workspace xcstrings-translator [command] [options]

# Example:
docker run -v $(pwd):/workspace -w /workspace xcstrings-translator google --api-key "your-key" --input "Localizable.xcstrings" --target-languages "zh-Hans"
```

## GitHub Packages Release

The Docker image is automatically built and pushed to GitHub Container Registry when you push to the main branch or create a tag.

### Manual Release Process

1. **Build and tag the image:**
```bash
# Get the current version from cmd/root.go or create a tag
VERSION=$(grep 'Version = ' cmd/root.go | cut -d'"' -f2)
docker build -t ghcr.io/fdddf/xcstrings-translator:${VERSION} .
docker tag ghcr.io/fdddf/xcstrings-translator:${VERSION} ghcr.io/fdddf/xcstrings-translator:latest
```

2. **Login to GitHub Container Registry:**
```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
```

3. **Push the image:**
```bash
docker push ghcr.io/fdddf/xcstrings-translator:${VERSION}
docker push ghcr.io/fdddf/xcstrings-translator:latest
```

### Automated Release with GitHub Actions

The workflow in `.github/workflows/docker-release.yml` automatically:
- Builds the Docker image on pushes to main branch
- Tags images with branch names, PR numbers, semantic versions, and commit SHAs
- Pushes to GitHub Container Registry
- Runs security scanning with Trivy

#### Workflow Triggers:
- Push to `main` or `master` branches
- Push of version tags (`v*`)
- Pull requests to `main`

### Using the Released Image

Pull and run the latest image:

```bash
docker pull ghcr.io/fdddf/xcstrings-translator:latest
docker run ghcr.io/fdddf/xcstrings-translator:latest --help
```

## Multi-platform Support

The GitHub Actions workflow builds for both `linux/amd64` and `linux/arm64` architectures.

## Security Features

- Runs as non-root user (UID 65532)
- Minimal Alpine base image
- Health checks built-in
- Built with security scanning via Trivy

## Build Notes

- CGO disabled for better portability
- Uses system users/groups for container security
- CA certificates updated in final image
- Health check uses shell command format for reliability
