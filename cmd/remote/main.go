package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"golang.org/x/crypto/ssh"
)

func createSSHTunnel(user, privateKey string, serverAddr, localAddr string) error {
	signer, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshConn, err := ssh.Dial("tcp", serverAddr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to establish SSH connection: %w", err)
	}

	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on local address: %w", err)
	}

	go func() {
		for {
			localConn, err := listener.Accept()
			if err != nil {
				log.Printf("failed to accept local connection: %s", err)
				continue
			}

			remoteConn, err := sshConn.Dial("unix", "/var/run/docker.sock")
			if err != nil {
				log.Printf("failed to dial remote Docker socket: %s", err)
				localConn.Close()
				continue
			}

			go func() {
				defer localConn.Close()
				defer remoteConn.Close()
				io.Copy(localConn, remoteConn)
			}()

			go func() {
				defer localConn.Close()
				defer remoteConn.Close()
				io.Copy(remoteConn, localConn)
			}()
		}
	}()

	return nil
}

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
	serverAddr := os.Getenv("DOCKER_HOST")
	localAddr := "localhost:2375"
	user := os.Getenv("SSH_USER")
	privateKey := os.Getenv("SSH_PRIVATE_KEY")

	err := createSSHTunnel(user, privateKey, serverAddr, localAddr)
	if err != nil {
		log.Fatalf("Failed to create SSH tunnel: %s", err)
	}

	cli, err := client.NewClientWithOpts(client.WithHost("tcp://localhost:2375"))
	if err != nil {
		log.Fatalf("Failed to create Docker client: %s", err)
	}

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
