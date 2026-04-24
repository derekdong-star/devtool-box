package service

import "strings"

type CookieService struct{}

func NewCookieService() *CookieService {
	return &CookieService{}
}

func (s *CookieService) Parse(cookie string) [][2]string {
	pairs := make([][2]string, 0)
	for _, part := range strings.Split(cookie, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		for _, sub := range strings.Split(part, "&") {
			sub = strings.TrimSpace(sub)
			if sub == "" {
				continue
			}
			kv := strings.SplitN(sub, "=", 2)
			if len(kv) == 2 {
				pairs = append(pairs, [2]string{kv[0], kv[1]})
			} else {
				pairs = append(pairs, [2]string{sub, ""})
			}
		}
	}
	return pairs
}
