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

	"github.com/Candratama/tamagosh/internal/config"
)

type Entry struct {
	Name    string
	IsDir   bool
	Size    int64
	ModTime time.Time
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
		out = append(out, Entry{Name: fi.Name(), IsDir: fi.IsDir(), Size: fi.Size(), ModTime: fi.ModTime()})
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
	return c.DownloadProgress(remotePath, localPath, nil)
}

func (c *Client) Upload(localPath, remotePath string) error {
	return c.UploadProgress(localPath, remotePath, nil)
}

func (c *Client) DownloadProgress(remotePath, localPath string, onBytes func(int64)) error {
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
	r := &progressReader{r: src, cb: onBytes}
	_, err = io.Copy(dst, r)
	return err
}

func (c *Client) UploadProgress(localPath, remotePath string, onBytes func(int64)) error {
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
	r := &progressReader{r: src, cb: onBytes}
	_, err = io.Copy(dst, r)
	return err
}

func (c *Client) RemoteSize(path string) (int64, error) {
	info, err := c.sftp.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (c *Client) Mkdir(path string) error {
	return c.sftp.Mkdir(path)
}

func (c *Client) Rename(oldPath, newPath string) error {
	return c.sftp.Rename(oldPath, newPath)
}

func (c *Client) Stat(path string) (Entry, error) {
	info, err := c.sftp.Stat(path)
	if err != nil {
		return Entry{}, err
	}
	return Entry{Name: info.Name(), IsDir: info.IsDir(), Size: info.Size(), ModTime: info.ModTime()}, nil
}

func (c *Client) Chmod(path string, mode os.FileMode) error {
	return c.sftp.Chmod(path, mode)
}

func (c *Client) Walk(root string, fn func(path string, isDir bool, size int64) error) error {
	return walkRemote(c.sftp, root, fn)
}

func walkRemote(sc *sftp.Client, dir string, fn func(string, bool, int64) error) error {
	info, err := sc.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fn(dir, false, info.Size())
	}
	if err := fn(dir, true, 0); err != nil {
		return err
	}
	entries, err := sc.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		sub := dir
		if !strings.HasSuffix(sub, "/") {
			sub += "/"
		}
		sub += e.Name()
		if e.IsDir() {
			if err := walkRemote(sc, sub, fn); err != nil {
				return err
			}
		} else {
			if err := fn(sub, false, e.Size()); err != nil {
				return err
			}
		}
	}
	return nil
}

type progressReader struct {
	r  io.Reader
	cb func(int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 && p.cb != nil {
		p.cb(int64(n))
	}
	return n, err
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
