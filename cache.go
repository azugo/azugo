package azugo

import (
	"azugo.io/azugo/cache"
)

func (a *App) initCache() error {
	if a.cache != nil {
		return nil
	}

	conf := a.Config().Cache
	opts := []cache.CacheOption{
		conf.Type,
	}
	if conf.TTL > 0 {
		opts = append(opts, cache.DefaultTTL(conf.TTL))
	}
	if len(conf.ConnectionString) != 0 {
		opts = append(opts, cache.ConnectionString(conf.ConnectionString))
	}
	if len(conf.Password) != 0 {
		opts = append(opts, cache.ConnectionPassword(conf.Password))
	}
	if len(conf.KeyPrefix) != 0 {
		opts = append(opts, cache.KeyPrefix(conf.KeyPrefix))
	}
	a.cache = cache.New(opts...)
	return nil
}

func (a *App) closeCache() {
	if a.cache == nil {
		return
	}
	a.cache.Close()
}

func (a *App) Cache() *cache.Cache {
	if a.cache == nil {
		if err := a.initCache(); err != nil {
			panic(err)
		}
	}
	return a.cache
}
