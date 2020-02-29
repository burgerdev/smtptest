package main

import (
  "context"
  "fmt"
  "net"
  "os"
  "strconv"
  "strings"
  "time"
)

type Response struct {
  Code int
  Msg string
}

func (r Response) IsOK() bool { return 200 <= r.Code && r.Code < 400  }

func Parse(data []byte) (*Response, error) {
  subs := strings.SplitN(string(data), " ", 2)
  if len(subs) < 2 {
    return nil, fmt.Errorf("could not parse response %q", string(data))
  }
  code, err := strconv.Atoi(subs[0])
  if err != nil {
    return nil, fmt.Errorf("could not parse response code %q", subs[0])
  }
  return &Response{Code: code, Msg: subs[1]}, nil
}

func Exchange(_ context.Context, c net.Conn, data []byte) ([]byte, error) {
  // TODO: respect the context deadline!
  _, err := c.Write(data)
  if err != nil {
    return nil, err
  }
  buf := make([]byte, 1024)
  n, err := c.Read(buf)
  if err != nil {
    return nil, err
  }
  return buf[:n], nil
}

func testMail() string {
  date := fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z))
  return (date + "Subject: Hello, World!\r\nFrom: no-reply@test.burgerdev.de\r\n\r\nHi there!\r\n.\r\n")
}

func main() {
  ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
  defer cancel()

  dialer := &net.Dialer{}
  conn, err := dialer.DialContext(ctx, "tcp", "mail.burgerdev.de:smtp")
  if err != nil {
    fmt.Fprintf(os.Stderr, "%v\n", err)
    os.Exit(2)
  }

  msg := []string{
    "",
    "HELO test.burgerdev.de\r\n",
    "MAIL FROM: no-reply@test.burgerdev.de\r\n",
    "RCPT TO: fritz_smtptest@burgerdev.de\r\n",
    "DATA\r\n",
    testMail(),
    "QUIT\r\n",
  }

  for _, m := range msg {
    fmt.Printf("client> %s", m)
    if m == "" {
      fmt.Printf("\n")
    }
    ans, err := Exchange(ctx, conn, []byte(m))
    if err != nil {
      fmt.Fprintf(os.Stderr, "%v\n", err)
      os.Exit(2)
    }
    fmt.Printf("server> %s", ans)
    resp, err := Parse(ans)
    if err != nil {
      fmt.Fprintf(os.Stderr, "%v\n", err)
      os.Exit(2)
    }
    if !resp.IsOK() {
      fmt.Fprintf(os.Stderr, "%+v\n", *resp)
      os.Exit(2)
    }
  }
}
