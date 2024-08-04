package mars

import (
	"net/url"
	"path"
)

func (s *Service) GetPlayerUrl(playerId string) string {
	//https://mars.blockthem.xyz/player?id=p643a7f4ae170
	reqUrl := *s.baseURL
	reqUrl.Path = path.Join(reqUrl.Path, "player")
	v := url.Values{}
	v.Set("id", playerId)
	reqUrl.RawQuery = v.Encode()

	return reqUrl.String()
}
