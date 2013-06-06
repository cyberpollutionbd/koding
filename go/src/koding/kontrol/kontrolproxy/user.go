package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bradfitz/gomemcache/memcache"
	"koding/tools/db"
	"koding/virt"
	"labix.org/v2/mgo/bson"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

const CACHE_TIMEOUT_VM = 60 //seconds

type UserInfo struct {
	Username    string
	Servicename string
	Key         string
	FullUrl     string
	DomainMode  string
	DomainName  string
	IP          string
	Country     string
	Target      *url.URL
	Redirect    bool
}

func NewUserInfo(username, servicename, key, fullurl, mode, domainname string) *UserInfo {
	return &UserInfo{
		Username:    username,
		Servicename: servicename,
		Key:         key,
		FullUrl:     fullurl,
		DomainMode:  mode,
		DomainName:  domainname,
	}
}

func populateUser(outreq *http.Request) (*UserInfo, error) {
	user, err := parseDomain(outreq.Host)
	if err != nil {
		return nil, err
	}

	host, err := user.populateIP(outreq.RemoteAddr)
	if err != nil {
		fmt.Println(err)
	} else {
		user.populateCountry(host)
	}

	err = user.populateTarget()
	if err != nil {
		return nil, err
	}

	_, ok := user.validate()
	if !ok {
		return nil, errors.New("not validated user")
	}

	fmt.Printf("--\nconnected user information %v\n", user)
	return user, nil
}

func (u *UserInfo) populateIP(remoteAddr string) (string, error) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		fmt.Printf("could not split host and port: %s", err.Error())
		return "", err
	}
	u.IP = host
	return host, nil
}

func (u *UserInfo) populateCountry(host string) {
	if geoIP != nil {
		loc := geoIP.GetLocationByIP(host)
		if loc != nil {
			u.Country = loc.CountryName
		}
	}
}

func parseDomain(host string) (*UserInfo, error) {
	switch counts := strings.Count(host, "-"); {
	case counts == 0:
		// otherwise lookup to our list of domains
		userInfo, err := lookupDomain(host)
		if err != nil {
			return nil, err
		}

		return userInfo, nil

	case counts == 1:
		// host is in form {name}-{key}.kd.io, used by koding
		partsFirst := strings.Split(host, ".")
		firstSub := partsFirst[0]

		partsSecond := strings.Split(firstSub, "-")
		servicename := partsSecond[0]
		key := partsSecond[1]

		return NewUserInfo("koding", servicename, key, "", "internal", ""), nil
	case counts > 1:
		// host is in form {name}-{key}-{username}.kd.io, used by users
		partsFirst := strings.Split(host, ".")
		firstSub := partsFirst[0]

		partsSecond := strings.SplitN(firstSub, "-", 3)
		servicename := partsSecond[0]
		key := partsSecond[1]
		username := partsSecond[2]

		return NewUserInfo(username, servicename, key, "", "internal", ""), nil
	default:
		return &UserInfo{}, errors.New("no data available for proxy")
	}

}

func lookupDomain(domainname string) (*UserInfo, error) {
	d := strings.SplitN(domainname, ".", 2)[1]
	if d == "kd.io" {
		vmName := strings.SplitN(domainname, ".", 2)[0]
		return NewUserInfo(vmName, "", "", "", "vm", domainname), nil
	}

	domain, err := proxyDB.GetDomain(domainname)
	if err != nil || domain.Domainname == "" {
		return &UserInfo{}, fmt.Errorf("no domain lookup keys found for host '%s'", domainname)

	}

	return NewUserInfo(domain.Username, domain.Servicename, domain.Key, domain.FullUrl, domain.Mode, domainname), nil
}

func lookupRabbitKey(username, servicename, key string) string {
	var rabbitkey string

	res, err := proxyDB.GetKey(username, servicename, key)
	if err != nil {
		fmt.Printf("no rabbitkey available for user '%s' in the db. disabling rabbit proxy\n", username)
		return rabbitkey
	}

	if len(res) >= 1 {
		fmt.Printf("round-robin is disabled for rabbit proxy %s\n", username)
		return rabbitkey
	}

	return res[0].RabbitKey
}

func (u *UserInfo) populateTarget() error {
	var hostname string
	var err error

	username := u.Username
	servicename := u.Servicename
	key := u.Key

	switch u.DomainMode {
	case "direct":
		u.Target, err = url.Parse("http://" + u.FullUrl)
		if err != nil {
			return err
		}
		return nil
	case "vm":
		var vm virt.VM
		mcKey := username + "kontrolproxyvm"
		it, err := memCache.Get(mcKey)
		if err != nil {
			fmt.Println("got vm ip from mongodb")
			if err := db.VMs.Find(bson.M{"name": username}).One(&vm); err != nil {
				u.Target, _ = url.Parse("http://www.koding.com/notfound.html")
				u.Redirect = true
				return nil
			}
			if vm.IP == nil {
				u.Target, _ = url.Parse("http://www.koding.com/notactive.html")
				u.Redirect = true
				return nil
			}
			u.Target, err = url.Parse("http://" + vm.IP.String())
			if err != nil {
				return err
			}

			data, err := json.Marshal(u.Target)
			if err != nil {
				fmt.Printf("could not marshall worker: %s", err)
			}

			memCache.Set(&memcache.Item{
				Key:        mcKey,
				Value:      data,
				Expiration: int32(CACHE_TIMEOUT_VM),
			})

			return nil
		}
		fmt.Println("got vm ip from memcached")
		err = json.Unmarshal(it.Value, &u.Target)
		if err != nil {
			fmt.Printf("unmarshall memcached value: %s", err)
		}

		return nil
	case "internal":
		break // internal is done below
	}

	keys, err := proxyDB.GetKeyList(username, servicename)
	if err != nil {
		return errors.New("no users availalable in the db. targethost not found")
	}

	lenKeys := len(keys)
	if lenKeys == 0 {
		return fmt.Errorf("no keys are available for user %s", username)
	} else {
		if key == "latest" {
			// get all keys and sort them
			listOfKeys := make([]int, lenKeys)
			i := 0
			for k, _ := range keys {
				listOfKeys[i], _ = strconv.Atoi(k)
				i++
			}
			sort.Ints(listOfKeys)

			// give precedence to the largest key number
			key = strconv.Itoa(listOfKeys[len(listOfKeys)-1])
		}

		_, ok := keys[key]
		if !ok {
			return fmt.Errorf("no key %s is available for user %s", key, username)
		}

		// use round-robin algorithm for each hostname
		for i, value := range keys[key] {
			currentIndex := value.CurrentIndex
			if currentIndex == i {
				hostname = value.Host
				for k, _ := range keys[key] {
					if len(keys[key])-1 == currentIndex {
						keys[key][k].CurrentIndex = 0 // reached end
					} else {
						keys[key][k].CurrentIndex = currentIndex + 1
					}
				}
				break
			}
		}
	}

	u.Target, err = url.Parse("http://" + hostname)
	if err != nil {
		return err
	}
	u.Redirect = false

	return nil
}

func checkWebsocket(req *http.Request) bool {
	conn_hdr := ""
	conn_hdrs := req.Header["Connection"]
	if len(conn_hdrs) > 0 {
		conn_hdr = conn_hdrs[0]
	}

	upgrade_websocket := false
	if strings.ToLower(conn_hdr) == "upgrade" {
		upgrade_hdrs := req.Header["Upgrade"]
		if len(upgrade_hdrs) > 0 {
			upgrade_websocket = (strings.ToLower(upgrade_hdrs[0]) == "websocket")
		}
	}

	return upgrade_websocket
}

func (u *UserInfo) validate() (string, bool) {
	res, err := proxyDB.GetRule(u.DomainName)
	if err != nil {
		return fmt.Sprintf("no rule available for servicename %s\n", u.Username), true
	}

	return validator(res, u).IP().Country().Check()
}

func logDomainStat(name string) {
	err := proxyDB.AddDomainStat(name)
	if err != nil {
		fmt.Printf("could not add statistisitcs for %s\n", err.Error())
	}
}

func logProxyStat(name, country string) {
	err := proxyDB.AddProxyStat(name, country)
	if err != nil {
		fmt.Printf("could not add statistisitcs for %s\n", err.Error())
	}
}
