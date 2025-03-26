package global

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/deatil/go-cryptobin/cryptobin/crypto"
	"github.com/spf13/viper"
)

var (
	Viper *viper.Viper //读取配置
)

func InitConfig() {
	Viper = viper.New()
	cwd, _ := os.Getwd()
	Viper.SetConfigFile(fmt.Sprintf("%s/%s", cwd, "config.yaml"))
	if err := Viper.ReadInConfig(); err != nil {
		log.Fatal("❌ 读取配置失败:", err)
	}
	Viper.Set("key.privateKey", GetPrivateKey())

	log.Println("✔️ 初始化配置成功")
}

func GetPrivateKey() string {
	pwd := []byte(strings.TrimRight(os.Getenv("PASSWORD"), " \t\n\r"))
	if len(pwd) != 16 {
		log.Fatal("❌ 获取密码失败！！获取私钥失败")
	}
	privateKey := Viper.GetString("key.privateKey")
	if privateKey == "" {
		log.Fatal("❌ 获取加密私钥失败")
	}
	key := crypto.FromBase64String(privateKey).SetKey(string(pwd)).SetIv(string(pwd)).Aes().CBC().PKCS7Padding().Decrypt().ToString()
	if len(key) != 64 {
		log.Fatal("❌ 解密私钥失败")
	}
	return key
}

// 加密方法
func Encode(data, password string) string {
	cypten := crypto.FromString(data).SetKey(password).SetIv(password).Aes().CBC().PKCS7Padding().Encrypt().ToBase64String()
	return cypten
}
