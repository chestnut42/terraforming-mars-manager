package mars

import (
	"net/url"
	"path"
)

func (s *Service) GetPlayerUrl(playerId string) string {
	//https://terraforming-mars.herokuapp.com/player?id=p643a7f4ae170
	reqUrl := *s.cfg.PublicBaseURL
	reqUrl.Path = path.Join(reqUrl.Path, "player")
	v := url.Values{}
	v.Set("id", playerId)
	reqUrl.RawQuery = v.Encode()

	return reqUrl.String()
}
