# GCLOUD Provider for DevPod

[![Join us on Slack!](docs/static/media/slack.svg)](https://slack.loft.sh/) [![Open in DevPod!](https://devpod.sh/assets/open-in-devpod.svg)](https://devpod.sh/open#https://github.com/loft-sh/devpod-provider-gcloud)

## Building and Creating a Release

### 1. Build Binaries for All Platforms

```sh
GOOS=linux GOARCH=amd64 go build -o devpod-provider-gcloud-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o devpod-provider-gcloud-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -o devpod-provider-gcloud-darwin-arm64
```

### 2. Generate Checksums

```sh
shasum -a 256 devpod-provider-gcloud-* > checksums.txt
cat checksums.txt
```

### 3. Create provider.yaml for Release

Copy `hack/provider/provider.yaml` and replace placeholders:

```sh
VERSION="v0.1.0"
REPO="flexbase-eng/devpod-provider-gcloud"

# Get checksums
CHECKSUM_LINUX_AMD64=$(shasum -a 256 devpod-provider-gcloud-linux-amd64 | cut -d' ' -f1)
CHECKSUM_DARWIN_AMD64=$(shasum -a 256 devpod-provider-gcloud-darwin-amd64 | cut -d' ' -f1)
CHECKSUM_DARWIN_ARM64=$(shasum -a 256 devpod-provider-gcloud-darwin-arm64 | cut -d' ' -f1)

# Create release provider.yaml
sed -e "s|##VERSION##|$VERSION|g" \
    -e "s|##CHECKSUM_LINUX_AMD64##|$CHECKSUM_LINUX_AMD64|g" \
    -e "s|##CHECKSUM_DARWIN_AMD64##|$CHECKSUM_DARWIN_AMD64|g" \
    -e "s|##CHECKSUM_DARWIN_ARM64##|$CHECKSUM_DARWIN_ARM64|g" \
    -e "s|loft-sh/devpod-provider-gcloud|$REPO|g" \
    hack/provider/provider.yaml > provider.yaml
```

### 4. Create GitHub Release

```sh
# Tag the release
git tag $VERSION
git push origin $VERSION

# Create release with gh CLI
gh release create $VERSION \
  -R $REPO \
  --title "$VERSION" \
  --notes "Release notes here" \
  devpod-provider-gcloud-linux-amd64 \
  devpod-provider-gcloud-darwin-amd64 \
  devpod-provider-gcloud-darwin-arm64 \
  provider.yaml
```

### 5. Use the Forked Provider

```sh
devpod provider add github.com/flexbase-eng/devpod-provider-gcloud \
  -o PROJECT=<project-id> \
  -o ZONE=<zone>
```

## Getting started

The provider is available for auto-installation using:

```sh
devpod provider add gcloud -o PROJECT=<project id to use> -o ZONE=<Google Cloud zone to create the VMs in>
devpod provider use gcloud
```

Option `PROJECT` must be set when adding the provider
(unless the project to be used is set as the current project in `gcloud`).

Option `ZONE` should be set when adding the provider.

Options can be set using `devpod provider set-options`, for example:

```sh
devpod provider set-options gcloud -o DISK_IMAGE=my-custom-vm-image
```

Be aware that authentication is obtained using `gcloud` CLI tool, take a look
[here](https://developers.google.com/accounts/docs/application-default-credentials)
for more information.

### Creating your first devpod workspace with gcloud

After the initial setup, just use:

```sh
devpod up .
```

You'll need to wait for the machine and workspace setup.

### Customize the VM Instance

This provides has the following options:

| NAME           | REQUIRED | DESCRIPTION                                                    | DEFAULT                                              |
|----------------|----------|----------------------------------------------------------------|------------------------------------------------------|
| DISK_IMAGE     | false    | The disk image to use.                                         | projects/cos-cloud/global/images/cos-101-17162-127-5 |
| DISK_SIZE      | false    | The disk size to use (GB).                                     | 40                                                   |
| MACHINE_TYPE   | false    | The machine type to use.                                       | c2-standard-4                                        |
| PROJECT        | true     | The project id to use.                                         |                                                      |
| ZONE           | true     | The google cloud zone to create the VM in. E.g. europe-west1-d |                                                      |
| NETWORK        | false    | The network id to use.                                         |                                                      |
| SUBNETWORK     | false    | The subnetwork id to use.                                      |                                                      |
| TAG            | false    | A tag to attach to the instance.                               | devpod                                               |
| SERVICE_ACCOUNT| false    | A service account to attach to instance


