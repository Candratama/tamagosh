package sftp

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/candratama/sshm/internal/config"
)

type Entry struct {
	Name  string
	IsDir bool
	Size  int64
}

type Client struct {
	ssh  *ssh.Client
	sftp *sftp.Client
}

func Connect(c config.Connection, password string) (*Client, error) {
	port := c.Port
	if port == 0 {
		port = 22
	}
	cfg := &ssh.ClientConfig{
		User:            c.User,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", c.Host, port)
	sc, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial: %w", err)
	}
	fc, err := sftp.NewClient(sc)
	if err != nil {
		sc.Close()
		return nil, fmt.Errorf("sftp open: %w", err)
	}
	return &Client{ssh: sc, sftp: fc}, nil
}

func (c *Client) Close() error {
	if c.sftp != nil {
		c.sftp.Close()
	}
	if c.ssh != nil {
		return c.ssh.Close()
	}
	return nil
}

func (c *Client) Home() (string, error) {
	return c.sftp.Getwd()
}

func (c *Client) List(dir string) ([]Entry, error) {
	infos, err := c.sftp.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(infos))
	for _, fi := range infos {
		out = append(out, Entry{Name: fi.Name(), IsDir: fi.IsDir(), Size: fi.Size()})
	}
	return out, nil
}

func (c *Client) Delete(remotePath string) error {
	info, err := c.sftp.Stat(remotePath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return c.sftp.RemoveDirectory(remotePath)
	}
	return c.sftp.Remove(remotePath)
}

func (c *Client) Download(remotePath, localPath string) error {
	src, err := c.sftp.Open(remotePath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func (c *Client) Upload(localPath, remotePath string) error {
	src, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := c.sftp.Create(remotePath)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func Parent(p string) string {
	if p == "" || p == "/" {
		return "/"
	}
	p = strings.TrimRight(p, "/")
	parent := path.Dir(p)
	if parent == "." || parent == "" {
		return "/"
	}
	return parent
}

func Join(dir, name string) string {
	dir = strings.TrimRight(dir, "/")
	name = strings.TrimLeft(name, "/")
	if dir == "" {
		return "/" + name
	}
	return dir + "/" + name
}
