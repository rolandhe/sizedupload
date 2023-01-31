package sizedupload

import (
	"context"
	"github.com/rolandhe/sizedupload/upctx"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"strings"
	"sync"
)

const globalName = "global"

var loader = &struct {
	once     sync.Once
	provider SizeProvider
	err      error
}{}

func NewConfSizeProvider() (SizeProvider, error) {
	return NewConfSizeProviderByConfFile("./conf/sizeconfig.yml")
}
func NewConfSizeProviderByConfFile(confFile string) (SizeProvider, error) {
	loader.once.Do(func() {
		cp := &configProvider{
			map[string]*confMeta{},
		}
		loader.err = load(cp.metaMap, confFile)
		if loader.err == nil {
			loader.provider = cp
		}
	})
	return loader.provider, loader.err
}

func load(metaMap map[string]*confMeta, confFile string) error {
	buff, err := os.ReadFile(confFile)
	if err != nil {
		log.Println(err)
		return err
	}
	config := YamlConfig{}
	err = yaml.Unmarshal(buff, &config)
	if err != nil {
		log.Println(err)
		return err
	}

	metaMap[globalName] = &confMeta{globalName, "", config.Global}
	for _, v := range config.Configs {
		meta := &confMeta{getBizNameByUrl(v.BizLine), v.BizLine, v.MaxLength}
		metaMap[meta.url] = meta
		if v.Subs == nil {
			continue
		}
		for _, sub := range v.Subs {
			meta := &confMeta{getBizNameByUrl(sub.BizLine), sub.BizLine, sub.MaxLength}
			metaMap[meta.url] = meta
		}
	}
	return nil
}

type confMeta struct {
	bizName string
	url     string
	limit   int64
}
type configProvider struct {
	metaMap map[string]*confMeta
}

type BizLineConf struct {
	BizLine   string `yaml:"bizLine,omitempty"`
	MaxLength int64  `yaml:"maxLength,omitempty"`
}

type BizConf struct {
	BizLine   string        `yaml:"bizLine,omitempty"`
	MaxLength int64         `yaml:"maxLength,omitempty"`
	Subs      []BizLineConf `yaml:"subs,omitempty"`
}

type YamlConfig struct {
	Global  int64     `yaml:"global,omitempty"`
	Configs []BizConf `yaml:"configs,omitempty"`
}

func (conf *configProvider) GetSize(url string, ctx context.Context) int64 {
	limit := getLimitCore(conf.metaMap, url)
	UploadConfig.LogInfo("tid=%s,get size %d for %s", upctx.GetTraceId(ctx), limit, url)
	return limit
}

func getLimitCore(metaMap map[string]*confMeta, url string) int64 {
	if url == "" {
		return metaMap[globalName].limit
	}
	meta, ok := metaMap[url]
	if ok {
		return meta.limit
	}
	meta, ok = metaMap[getBizNameByUrl(url)]
	if ok {
		return meta.limit
	}
	return metaMap[globalName].limit
}

func getBizNameByUrl(url string) string {
	items := strings.Split(url, "/")

	for _, v := range items {
		if v != "" {
			return v
		}
	}
	return ""
}
