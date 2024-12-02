# Docker Environment

## Building and running your application

To build and run your application, you'll need to install Docker.
Customize the [Dockerfile](Dockerfile) and the [compose](compose.yaml) file to your liking, then, when you're ready, start your application by running:
`docker compose up --build`.

Your application will be available at <http://localhost:8000>.

### Deploying your application to the cloud

First, build your image, e.g.: `docker build -t app .`.

If your cloud uses a different CPU architecture than your development
machine (e.g., you are on a Mac M1 and your cloud provider is amd64),
you'll want to build the image for that platform, e.g.:

`docker build --platform=linux/amd64 -t app .`.

Then, push it to your registry, e.g. `docker push my_registry.com/app`.

Consult Docker's [getting started](https://docs.docker.com/go/get-started-sharing/)
docs for more detail on building and pushing.

### References

* [Docker's Go guide](https://docs.docker.com/language/golang/)
