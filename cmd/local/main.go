package main

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

func pullImage(cli *client.Client, imageName string) error {
	log.Printf("pulling the image: image = %s\n", imageName)
	reader, err := cli.ImagePull(context.Background(), imageName, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	io.Copy(os.Stdout, reader)
	return nil
}

func startContainer(cli *client.Client, imageName string) (string, error) {
	log.Printf("creating a container: image = %s\n", imageName)
	resp, err := cli.ContainerCreate(context.Background(), &container.Config{
		Image: imageName,
	}, nil, nil, nil, "")
	if err != nil {
		return "", err
	}

	log.Println("Starting the container...")
	if err := cli.ContainerStart(context.Background(), resp.ID, container.StartOptions{}); err != nil {
		return "", err
	}

	return resp.ID, nil
}

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal(err)
	}
	cli.NegotiateAPIVersion(context.Background())

	imageName := "hello-world:latest"

	if err := pullImage(cli, imageName); err != nil {
		log.Fatal(err)
	}

	containerID, err := startContainer(cli, imageName)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("container started: containerID = %s\n", containerID)
}
