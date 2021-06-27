package auth

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	xhttp "github.com/Jarijaas/go-tls-exposed/http"
	xtls "github.com/Jarijaas/go-tls-exposed/tls"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	log "github.com/sirupsen/logrus"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
)

const GooglePubkey = "AAAAgMom/1a/v0lblO2Ubrt60J2gcuXSljGFQXgcyZWveWLEwo6prwgi3iJIZdodyhKZQrNWp5nKJ3srRXcUW+F1BD3baEVGcmEgqaLZUNBjm057pKRI16kB0YppeGx5qIQ5QjKzsR8ETQbKLNWgRY0QRNVz34kMJR3P/LgHax/6rmf5AAAAAwEAAQ=="

func parseKeyValues(r io.Reader) map[string]string {
	scanner := bufio.NewScanner(r)

	kvs :=  map[string]string{}

	for scanner.Scan() {
		row := scanner.Text()
		firstIdx := strings.Index(row, "=")
		key := row[:firstIdx]
		value := row[firstIdx + 1:]
		kvs[strings.ToLower(key)] = value
	}
	return kvs
}

// Encrypt creds using RSA and google pub key
// https://github.com/NoMore201/googleplay-api/blob/master/gpapi/googleplay.py
// If randSrc is nill, uses crypto/rand.Reader
func encryptCredentials(email string, password string, randSrc *io.Reader) (string, error) {

	if randSrc == nil {
		randSrc = &rand.Reader
	}

	pubKeyBin, err := base64.StdEncoding.DecodeString(GooglePubkey)
	if err != nil {
		return "", err
	}

	modulusLen := binary.BigEndian.Uint32(pubKeyBin)

	modulus := pubKeyBin[4:modulusLen + 4]

	offset := modulusLen + 4

	exponentLen := binary.BigEndian.Uint32(pubKeyBin[offset:offset + 4])

	exponentBytes := make([]byte, 4)
	copy(exponentBytes[4 - exponentLen:], pubKeyBin[offset + 4:])

	exponent := int(binary.BigEndian.Uint32(exponentBytes)) // 65537

	hash := sha1.New()
	hash.Write(pubKeyBin)
	digest := hash.Sum(nil)

	h := append([]byte{}, 0x00)
	h = append(h, digest[:4] ...)

	n := new(big.Int)
	n.SetBytes(modulus)

	pubKey := &rsa.PublicKey{
		N: n,
		E: exponent,
	}

	msg := append([]byte(email), 0x00)
	msg = append(msg, []byte(password) ...)

	ciphertext, err := rsa.EncryptOAEP(sha1.New(), *randSrc, pubKey, msg, nil)
	if err != nil {
		return "", err
	}

	final := append(h, ciphertext ...)

	return base64.URLEncoding.EncodeToString(final), nil
}

// Create http client that bypasses the TLS fingerprint check
// Uses modified tls package, so may be insecure
// Therefore, use this only when necessary
func createXTLSHttpClient() *xhttp.Client {
	conf := &xtls.Config{
		CipherSuites: []uint16{
			0x1302,			0x1303,			0x1301,			0xc02c,
			0xc030,			0xc02b,			0xc02f,			0xcca9,
			0xcca8,			0x00a3,			0x009f,			0x00a2,
			0x009e,			0xccaa,			0xc0af, 		0xc0ad,
			0xc024,			0xc028,			0xc00a, 		0xc014,
			0xc0a3,			0xc09f,			0x006b, 		0x006a,
			0x0039,			0x0038,			0xc0ae,			0xc0ac,
			0xc023,			0xc027,			0xc009,			0xc013,
			0xc0a2,			0xc09e,			0x0067,			0x0040,
			0x0033,			0x0032,			0x009d,			0x009c,
			0xc0a1,			0xc09d,			0xc0a0,			0xc09c,
			0x003d,			0x003c,			0x0035, 		0x002f,
			0x00ff,
		},
		TicketSupported: true,
		PskModes: []uint8{xtls.PskModeDHE},
		SupportedVersions: []uint16{xtls.VersionTLS13, xtls.VersionTLS12},
		SupportedSignatureAlgorithms: []xtls.SignatureScheme{
			0x0403, 0x0503, 0x0603, 0x0807, 0x0808, 0x0809, 0x080a, 0x080b, 0x0804, 0x0805,
			0x0806, 0x0401, 0x0501, 0x0601, 0x0303, 0x0301, 0x0302, 0x0402, 0x0502, 0x0602,
		},
		OscpStapling: true,
		Scts: true,
		CompressionMethods: []uint8{xtls.CompressionNone},
		SecureRenegotiationSupported: false,
		ClientHelloVersion: xtls.VersionTLS12,
		SupportedPoints: []uint8{xtls.PointFormatUncompressed, 1, 2},
		SupportedCurves: []xtls.CurveID{0x001d, 0x0017, 0x001e, 0x0019, 0x0018},
		Extensions: []uint16{
			xtls.ExtensionServerName, xtls.ExtensionSupportedPoints, xtls.ExtensionSupportedCurves,
			xtls.ExtensionSessionTicket, xtls.ExtensionEncryptThenMac, xtls.ExtensionExtendedMasterSecret,
			xtls.ExtensionSignatureAlgorithms, xtls.ExtensionSupportedVersions, xtls.ExtensionSignatureAlgorithmsCert,
			xtls.ExtensionPSKModes, xtls.ExtensionKeyShare,
		},
	}

	transport := &xhttp.Transport{
		TLSClientConfig:        conf,
	}

	return &xhttp.Client{Transport: transport}
}

func getSubToken(masterToken string) (string, error) {

	params := url.Values{}
	params.Set("service", "androidmarket")
	params.Set("app", "com.android.vending")
	params.Set("Token", masterToken)
	/*params.Set("token_request_options", "CAA4AQ==")
	params.Set("system_partition", "1")
	params.Set("_opt_is_called_from_account_manager", "1")*/


	params.Set("source", "android")
	params.Set("client_sig", "38918a453d07199354f8b19af05ec6562ced5788")
	params.Set("callerSig", "38918a453d07199354f8b19af05ec6562ced5788")
	params.Set("lang", "fi")
	params.Set("device_country", "fi")
	params.Set("has_permission", "1")

	httpClient := createXTLSHttpClient()

	req, err := xhttp.NewRequest("POST", AuthURL, strings.NewReader(params.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	/*req.Header.Set("device", gsfId)
	req.Header.Set("app", "com.android.vending")*/

	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}

	kvs := parseKeyValues(res.Body)

	errorDesc, has := kvs["error"]
	if has {
		return "", fmt.Errorf("google auth API returned error: %s", errorDesc)
	}

	log.Debugf("Round token results: %v", kvs)
	return kvs["auth"], nil
}

var ExecAllocatorOptions = [...]chromedp.ExecAllocatorOption{
	chromedp.NoFirstRun,
	chromedp.NoDefaultBrowserCheck,

	// After Puppeteer's default behavior.
	chromedp.Flag("disable-background-networking", true),
	chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
	chromedp.Flag("disable-background-timer-throttling", true),
	chromedp.Flag("disable-backgrounding-occluded-windows", true),
	chromedp.Flag("disable-breakpad", true),
	chromedp.Flag("disable-client-side-phishing-detection", true),
	chromedp.Flag("disable-default-apps", true),
	chromedp.Flag("disable-dev-shm-usage", true),
	chromedp.Flag("disable-extensions", true),
	chromedp.Flag("disable-features", "site-per-process,Translate,BlinkGenPropertyTrees"),
	chromedp.Flag("disable-hang-monitor", true),
	chromedp.Flag("disable-ipc-flooding-protection", true),
	chromedp.Flag("disable-popup-blocking", true),
	chromedp.Flag("disable-prompt-on-repost", true),
	chromedp.Flag("disable-renderer-backgrounding", true),
	chromedp.Flag("disable-sync", true),
	chromedp.Flag("force-color-profile", "srgb"),
	chromedp.Flag("metrics-recording-only", true),
	chromedp.Flag("safebrowsing-disable-auto-update", true),
	chromedp.Flag("enable-automation", true),
	chromedp.Flag("password-store", "basic"),
	chromedp.Flag("use-mock-keychain", true),
}


func doBrowserVerification(url string) (string, error) {
	verificationCompleted := make(chan string, 1)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), ExecAllocatorOptions[:]...)
	defer cancel()

	// also set up a custom logger
	taskCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	chromedp.ListenTarget(taskCtx, func(ev interface{}) {
		switch ev := ev.(type) {

		case *network.EventResponseReceived:
			resp := ev.Response

			if len(resp.Headers) != 0 {
				cookies := resp.Headers["set-cookie"]
				if cookies != nil {
					rawCookies := cookies.(string)

					header := http.Header{}
					header.Add("Cookie", rawCookies)
					req := http.Request{Header: header}
					for _, cookie := range req.Cookies() {
						if cookie.Name == "oauth_code" {
							verificationCompleted <- cookie.Value
						}
					}
				}
			}
		}
	})

	err := chromedp.Run(taskCtx, chromedp.Navigate(url))
	if err != nil {
		return "", err
	}

	return <-verificationCompleted, nil
}

func getMasterTokenFromOAuthCode(email string, gsfId string, code string) (string, error) {
	httpClient := http.Client{}

	body := "androidId="+ gsfId +"&lang=en-US&google_play_services_version=212116028&sdk_version=28&device_country=fi&Email=" + url.QueryEscape(email) +
			"&build_product=OnePlus3&build_brand=OnePlus&Token=oauth2_" + url.QueryEscape(code) +"&"+
			"build_fingerprint=OnePlus%2FOnePlus3%2FOnePlus3T%3A9%2FPKQ1.181203.001%2F1911042108%3Auser%2Frelease-keys&build_device=OnePlus3T&service=ac2dm&"+
			"get_accountid=1&ACCESS_TOKEN=1&callerPkg=com.google.android.gms&add_account=1&"+
			"droidguard_results=CgYF2l7hudYSgC8KBiA4bc5ukdIQQAABf_syPB_ewtEA6ndQDfWyb5IATmYTeOdpo3wEerS0wsSdyYYAtMmfjcLRAZ8BfaoOp5IB7EwACpFXGHHy09baB_kVAOuhCBKdLrROz9cr0C1jpqN3W6hfsigoJI97vBdtr0f1uil3_-23UA81oyRewIV4ezPUPzmhyVXOs3MJCLK0cnpsF76qqsPopml3pfUmpDMwVWKtHAVN-ZvpIRj3lJj3qpS3Zh5MDgmAr6w-z-lU9Sg6otzglr_t749qlmBRz9DVU54lAnm1SN2B9UmGavjoW6rS_E7pBCk8iE2Q-k4hMBKptkI1SYlyX_uUTA8iqQvx6MMeLMzL5Q5b-lOYhCXpov5jdXi0nDZKpxky6E4QQtTyZe6XGNii9c3HY4nGUrdar-OyjnOtsdjPzIB6pzCcmOc2G-jJScEtPJ_1H7O8Qakst0DlF-K3vfMJ3JJubAOXk1t6J6TnpHr8HUvCRa-H1Qgz0ENacn_z5rYKPizMa7q1EBnUEPfzOzFv1L7Tva-s30CmpjA2lPbdM6OgJ_EiuzBXvgUIwQrpKEBZx0AjYSgcaZ8lZl-PtQgu72cQzxLLpAtLQFDVMlMhcGjHzhXTMkwGisFv_jy_77kZHogJm5tqLYk93aIudI18EMEb0pXK1VUstHf9j84O_Lo_ONByHYrqDlsl2BFnliD7KvD24su09-KEVGZMy7nDwjVpkx2qHbYnxu9KxORvqdKowrNKAiCsHe1Kcz_bVA054d9uf5sJ7Sjyn8KPE4GJfkpib4mHtfwHCCm--FfGXqRx7TSpULNqnbdS7CCGJclFuAz6k6ZkR-wV_beTdvgWtprtCxo--83ajYJSe5yDfvLnGJ8o3d6As_EkWCw-wTf03ZGSCBcJ9pdC7IwPN5tZKsWvwUNGzsxMDsbV8M-p3R5ODV6ww5oCv0hVzBx5Zuq6TcBHYKkJ-eCPt7hGlg3kxnX0aetsqoPFPCa56s3qkoc3pYBTSImPn-WeGcc6M9U2db5xIwsH8c8c1ykzA8m9gYfpsC_HMDTlb5zxp7Wxu3hTWbeQoi3IkPCBwTrUUa5abBecDii7YabxnmITiBT57yKUFLwE8XuQ5WidQCgoKlHrJXGSwC_zUCfnfVu8-YxResi_x8TE9jYvYP7SiWj8VvIPN08EXX7kHlJ2-QlVvnYKnG7oHW0iTBDQlotMNEV5ya_JAnEpYm-DiVl9hj8NpOvHGF3NI6yJokqgi1CkHH3s_x3NRmMkZsbLHTSibns8ipThOgMvx-FvJqLjqocbN9bO-GBhYtoKKpe1rCtJ7F5f3xzU8TZMeyiX0tqbW21CNpKEEtN7ww0_qEdxvajYl2d-tubtq6sCugBvlJqEDtRj_hW4-d2W9yursoIAu3xqlv7i5mchNUJ80GfGMoTyMrc2xuZD7XzzaQ_BlJecSF-YvrOZFEIvLv9gpAhMizwUuWni0hIR7ziT7mJFKXiuF6QLGiTtdoip63ZW8OxXQJxgBjForGONfvSRG1hyAVRV3nnEAY2Sg4iYrGfH1IwzjZW_A8bljai3XtJAmuPIR1OXOlsrxW2Qc5f8iwjWKjhsmW70ims-Ytg-u5ZPvd6O4Te33KJqKavuwGb8WZZ525XXJetZDa3qhSw2S5JN_3xzBcWO0r9EWQBljK1F8cKiWN7IVlADOFhpveWVzAHgU_yqEd3a__evtvbKl5tBcDWRgnrvNJED3AOj58wV24HxzF1dkrRBXxFrDXmRdVDspjenyMq-i295DPYAfsBuA6fdg84RMIDlOK2dEDoiiSko7w3gRwNQuDurmfia-HUbd2NX5efETHrfZpljBRVHfFt1twQPlSiHUMROi08wIZHbwq1Foi4ACuP_N_oOcNCGNogzTDGUTSkKhOCzusCA-s-5xt9rNGGrD-T5kTH0IoZzdvDsNTHaNGGMzPA0WSyZbqaDjtPJcMURDA8TBHOpSjJBYAxuWE5QQrK0DHqRiGNf9BlhaFmiiNuwprrDn4JRmHs_wb5IFixQ8sC30Z8h9NLpDMf4pYL2U2YL1ysOoElzwToxGVqSxKvQFABYw1cH1tJacBrRMNu2AGbTJgxCi61LTgsGaoXdU1aEP9jH7wR3n-rVmShHzbSslfWeuTHCvj3rz-cSjKS_MhD48Jel5SgIsy_i2DGgab0QHSeAw4Tk5L0DbDVydf9sdEVsi3C46gmHHk6hK8-qI1aEA_22gCpULNCwDlvBQkm-_1keMKUg-epp-L_OkCpsFDLN6bKUhdou58-0aG3lWvyIxw2iV7MebCf_RkoNTaeQV098QiXhJKOIFF5FgP5dT2TYUk4HeRA1bNjgONJG8qvngzTrQw0n34iVccP5_J8HSUuTGbfrV9pv9EFsf9szn5IRgMWMeAPsK1Qp71ae_FLyU5hjiz9lrifeOUsmOWLiVLVraCnLlQe7KAy5BvR0uHjky_i9IGLySzDi15qFweLKOBs_RtXbO38dLIMJ50ECey73GVeqxLL0zjnfPxdQG0qjSaZZWz3ryRcPc6mf45wSPneGXmQ34uuB7sVA8x1NUlXcHpraxnBypGjPI4_K8CAs_9zuPE8tFtWviNLy8-FeXOSted-tLVd-cPq7NbYB-yY86QhylP8nCWGQyURAvbzZ2I8T9b81Consi5JZqaw0X2KCwha-fO_i5HaMkvPeJ8OXY26D7zj__0CwXmbpEk3l-OGgk4PkOAkGJ9xoWqi4gcQa2lDKeDu73b9h88C1wcpV395k4x7SF42xjynyiO3Y-spv6dUVGZxlhl1PvnaI-GkPkN0QoBKEqwmOlf9mv1N_L6xsTMJ0PFFh3uSfxvByySqJ33RSxSnidnYo21CAyVG2qrDXyd7Ocv_XoEe-5mZxvYyxaPsjNe8VABGn24R3u7EpIrp6dOJnP44pRAjUxGWJv-b7KOFN6LlrAgucdmzf86fe7S_0wmUHMElOxcYX8t68yrcQmmlydqL0CVvHkYm1-P71HNz-pcFid5t8IzRe9JZTuGBSxB8VAw7mmkfclCr1xMwnhNr2TUwjC9gxXe7Yy6Ab0JLJbPTnTSz9-PS5QJBpT7kRh1oLrt5OL2VWH6UOcoQcCCTi0QIgfTfRXJ83Z5SY9d8Eg8Wp92z9U7gXU6klYIvNcBhF6EdElUlRRTta48sHl_UQ1wtr3bn2tNH478FZfXTzvh-0H8als7Tbbnyk_tAoksbjwC3PIHuOxLR9BlOxNKQcB1fInV4BF6fmA9NiQDtmTOdhbycmF7lcDcfycBP3O6ugn5L4o-0_WXO0WIkZKEvXwjmIvqQUTg1rxB_5Q4tDCrtc1CznEECegCVnQoLqD1W-tFWtikiUbZ9BU_YBw2v7eKAqH46SIV-8o80RH1O1v3yyyv4SBV8aehtuto2kZJrnkBX0upo-2hOsVou3LcFr4wcZrSaRRRO0DsIExuejG99fQdndo8cBUf_opTQK56GK7CzTUrBXqC004ObUVGBAT9zGrN5uHd_VvpRK7Lvdxc8-cyXidkP2_Nij3CByrEzxySS5OOuTT98DYKTfvFquNeMSfCLK9wca4YZPSZcVdfGNGEzWI0n0T28pc5bJ3bLPkULT8iTZoxycZ6tre_2Frb3RaEukjzD26n6d2dzHthkR2s1dBofa3ucckW5E65CbsnGzsY3vIFo385O9XVnH2AbViRPMTfrSbHLeyUz1Ltsr5oO0klGdLNM84xmG-kKUzZ6Tn4v-k0iu9DO4WImE3yUbsrdLjW-DQxSkRTu7kQ111aMzIuOaVZ-aNKHytkR1GaDuXRCK5nBOA8YjRzprQWdZg_BvTB1jFtKJV0shuCHDaD9XNe0etThh0A_Cq86-kMatobEB6AGm7fD9A-gBj_iN0Pj_____AThrOEuYEr-bqKiM9Z3oKJgSusmzm620gZNwEtcXKcCqPmsL6vAht40DA8Sn7UJstp95b2meK4zKI7WjrSmI_87iNB_q05aYn_imgE57s5Vi2QKYE76sEbyvwkBsvthlKm3FyjycEHTlfxseDe_Rhi0Gza777GNp_4tyIeMxpuRt7eLYx2akQJ9iwmS7_xlXoKe1UqMmSXnMTVu8Dz7boxxbPCfbxDQnttL5-e4rmRGhg5N5znCzZ4fw1KDn2bgJlqYwAlNDs2v6lEWxW6p9ZfMKn0CZqxiQuJ6k6OCWNBd61Yyjog07TlZC4es8NB49gPL-XDYAfjRUqGLPlb1GkFt5l27MGK5ILNdx3Yk6IC0HXYK3JayiWbPxxOvVK8pGky1RB87cC8uTgfn0pVpFVjj8EsPpOkIId6WR2ScuFfjQyIsheKxPXzYhKBA7teRrfIcjSw70JRQvyVnjAGjBZgvw6a6wjQDEuQiD4ngBKeBVjTFeUqYcH3DktTi2QlicGKAAu7xko7twX7xjZhExFVryp76iNHr80HYqt-Jg7zNMoqcD5dJtJ9WOi7vb6W8TE3gprD6AuUN0TAzuT1uvE54NhnUCtW4e-ihebNaJ4vNK523O3-Cq-eO2Esd2puPmGq2XnglNJJzM5PgRdQDxqHfeWggv1pY4cBJDvLDHbDZsmQ6YSn005Y8nb34lRyyfhkZLSpuxYdoLgqOz1BisAPzHeEFcQ2PpqVChEw-m6aZLyIxix9KOc8goubLwful8biJ1_w89hyJ5Xf6gy51KpVmvKFgVURkJzjzzBxTyFUglECPSzDchjfc8pCCOGAd08YcNx8qZbLsgS9buEMZCrLq9xGi47jrhvWyimPFSkCBqvBRb7mXcNLrLydzjMyv_Tm2hoF0mRyN76p-WndWzDfciN3ii4cDwUO-Ts86VOAfnlgNVnk-4aY9oBsyZ-hU9XFkwwVVsEeQ1vbImnGvI4awQiN7dAAaD_DjqX2vZASeeF9tXQ5lLwbWathe24br9wiDAuoeVeLmM7P0MDsdpPE_GfW3N6MZPeRfCvMsDq-YSeYlE939fNHPQO3rP9kTggN6jc5c8ahc1YMFZUSHlGWw0_GnddJOn5jSBopAe_ylOFvdFKM3j7XC91x3KUCsw1HK8JJSESZr401u2rMg4g57FYIrQNCq-O3XiRvcrTjHpjrPUUvmM2A5ezmjnwEWEGURLCDsOWyyume4-AG79q5jqWloxGLHozSHrPAuHSJMwn2wzgnCd-Px5_b4JyOIAPWxay6Ca-ukbSLbl_acJM7zaDe6nlgsiIocGxGd6k7hzAoZ7gHZvsutX3GlqaLRF_ufMt7v1rnx2egz_uHnQMRTMFquA6TNBz5ZFEFWExbuAv6MuoyIzitf2DMyuPTTXwOqJyTfy0Xpr_Oy7JweOrxdr8rPK2xzdPK7b0zbwINl_gavLQwEtB5vSlHs5TDUnq-u_GEv_U_bAF2F9Jy9_nBiHzTx6VwJjx5U2-OjIa9x1g8v5gbTOBTf0qKF1O6FGiAWo_flWmTAR--hTqsXEaJV09e2fLAlj3Mzp_NqB-dskV-pklE_R_-vCg79SXz_FbUPwVOWu0oNTjuDQjevn9ge6_xonHfR75H_uPyAjpf2dqnFtr41IpcuHC3tW6ot0GN0PMpNgZWEQOwbeTIawR8aqjjDj8k5-766pSwW0KIv_GSxU1BxOodVz-afO01PGB-8mItLNs5Y7zX1_kdl7AUiunS-ivpbcJfZVpLi370RKLwtky_83AY3VwyXE-1AqxomJg8iTYrzCjBqMO1R4HZttblgTGCxDqlyOTKl_0ONTGX8ey_R4ARV4hTiWLt4NNJZt0c2UQ1s8bh-TDde0aPO_ZyfccaNBlkNzPW-Idi0ujBHRbwWLUMBEVgKaZVsmw-7P_PnkMboQwDst3uErWsob0wOkXVFcPuYADPXm7Bch8BhXA7tWoJMxn6yMpczEI5_EWP7Z5logT_xUtIprKR_xlRM6Yv2Cb4BZBsvJl37Hq47XFpGuoGU-hUgi9ODzf-edGU5bJn-eMHv8ejMIx6OpieokkfyeGgwnt8qGSTpD6Uaem-SKVoHvYkhUZV_goaimjbrG770g3PoSxQVCCilCW_dgioMKcL6B4YfWR-EH9AMIxtGcb_swX36hS-H4N9uV0SwFIc8U5oRqUF6lwiKcNqh1x7560i2cQngIXIi-o0ttLY1NtebgQGN6oESjEUvFcop2biR7dTwXVbFvmPKNGZIDEmQj4qch52iXdYub-3EkOSDa7QNxyYn8ALX6lJLJsdv4r9m1jY1_IwvXtWWfV7hU_5J5-RfhxxgelRtxxz8EWBBUTKzT-ywiMv2vcPB5y21HoRM-Zpf-sxfYeE_P30VlqqHGP-QphW77OjOszkwYGo_S6YwltVLZqmg5UBmduXvEWWE3PPpNMxSo8_-8x-phoZ3gn3WnKxhMtEZq3KxPFLR4dyaMunaO7edDXKUq8v78EoRPLd3VNjQfpUfEIyQ5HABRenOucQvM7apEtbMmjFBumeuulYBFYmYaIYnGII1yswzD2aXeEMBL6eG1dd_4U8Bk2y23BFGuY57e1GzFLBD-SEZkNy-b7qX-sn6kSuWn4qEu4e5EwDx57JvkZDDenj9rGUe7LtrQf4-gGkm-LzVJAMv1nMMmDOetXbF3FUmqmoGdPFDd9ZGQ2iU9adrL80XGuDc-slaS7uiThjGoTO0KqBfNPrhMO0u-3zmfwKgXz7i2XgsXtt5M-zmltsvaHj4_Mi7V5fh0ZCQ9xYW5iMm5QN06yWLqHVZl5SmMk4-TiqPqyd1NgHSFYUh_OM6aDhlkusk7UXYWma2GCdIRsqur8hjr1yuh9z-yRULFP1HjSJjPKb37jzmvl_OsQeGA6iRpni2Z0QCZSR09g9vHxibyqsCG7lPbXS-QKN_sf-WEwbsMgLcjhuPJYMGUG_8EKopBGVkz68pWPrD7w9xkLaxo7_voybQM5OAfj1TUgTqp8hLEw3bTD7eGBJPOSZ4HaRtg-dBQ80SaFP97Y0-P-Trt1SoDUcNze5RO3mfV1cZt1BYTLu56WgRNVtfVGBMrn22GhsrkfQE0Ll9IY51_c4h0CSfWWtYXqARFnOh1YebtVRvVOAQjypEea5NClrLDfRY3dCIm5rhO9r0HJXBW7PefjRY6IJbeABMM7VUyRsCHkc-k7mTHpCKsnZ08nx-nr5DVAaRULZRAQkD7lIwah8koGFA2AKprJtbqqoHBuB6loaj4v5qy0Z1lgfUfLFfJay3HqkI4TQG7CD3ZLensBIK4g6VMhd6XmQCU1uYt_MOFproyQYwg5oxxHSxOATgbMVeRZ-Gh_t0MeLN4JA8t3zQZJBy-K8u2o81dFM9HxUd3UNurHTMZkFwv5jTGDuBfVKXzNDKX976NIaH2mQccj1ksLlb4ITS7SWMRrVNQ4d0qrtat_feIb8smrTHa62CViS52YQfGjM3oSMeea_35XJgm7DbfaNm9nCpVNM3EEWFRUPUEkzgMbq2DhNSuINrdE9sa5J7PY-1sKCoXL9gq7cs_GDvC79WJ7xhBq2xydhT3yN640VSa2F1dQ1HreomJdzotIz3M9HCohtRulbLoi19fHtTW3YRhqrav61jiMDiH2nom6L5k2DLXJBrDDZiBggDFnN9rY4m15vQPkSvYjp-_wOT-5831F1pgvB5A8JYIR72vvJM3DQV7cHplKkyeV9HB5Oy2F2nV-l50rUYY7jnpVg1NSANvDhVgyuMcj-iYMaPAm-CVqp-U4-kDNbn15rqZuTBCms_8NumlGJ_xwiYTSJwtEgXLMA1LKGaJ5dJKbqrfzeIK2dNCZxrg_ZKWARHvd5LtqVMHJVAOtqD52notfSFGJyUHSMeKVEec9FM96Z3ToZrvg92KAeuRpf8rnV-6sRf7f6m5wWN9VLsWIIVD1AcNvUUmgprjGc_Kca8oEV6U0LGjL5weVMpSp2u-SpuOi76XZyMufn_C4H98M_czYQx0vQh8a9JsDQ0olsjP_26YopUqN5FHOSP3c-GUo8YkLrAp2HP1HMC4nblzj6oQIOCXb7P-dVlMIW3vbZUUWHzIrsJP8ab8AvL5yNe3GzlgGhYKAggBCgQIDBBpCgQIDRAACgQIDhADIgA&callerSig=38918a453d07199354f8b19af05ec6562ced5788"

	fmt.Println(body)

	req, err := http.NewRequest("POST", "https://android.googleapis.com/auth", strings.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}

	kvs := parseKeyValues(res.Body)

	log.Infof("%v", kvs)

	if token, has := kvs["token"]; has {
		return token, nil
	}
	return "", fmt.Errorf("OAuth response did not contain master token")
}

func getPlayStoreAuthSubToken(email string, encryptedPasswd string, gsfId string) (string, error) {

	params := url.Values{}
	params.Set("service", "androidmarket")
	// params.Set("app", "com.android.vending")

	params.Set("Email", email)
	params.Set("EncryptedPasswd", encryptedPasswd)
	params.Set("add_account", "1")

	/*params.Set("source", "android")
	params.Set("client_sig", "38918a453d07199354f8b19af05ec6562ced5788")
	params.Set("callerSig", "38918a453d07199354f8b19af05ec6562ced5788")
	params.Set("lang", "fi")
	params.Set("device_country", "fi")
	params.Set("has_permission", "1")*/

	httpClient := createXTLSHttpClient()

	req, err := xhttp.NewRequest("POST", AuthURL, strings.NewReader(params.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}

	kvs := parseKeyValues(res.Body)

	log.Debugf("%v", kvs)

	var masterToken string

	errorDesc, has := kvs["error"]
	if has {
		if kvs["info"] == "WebLoginRequired" {
			log.Infof("Verification required, url: %s\n", kvs["url"])
			log.Infof("Opening browser ...")

			oauthCode, err := doBrowserVerification(kvs["url"])
			if err != nil {
				return "", err
			}

			log.Infof("oauth code: %s", oauthCode)

			masterToken, err = getMasterTokenFromOAuthCode(email, oauthCode, gsfId)
			if err != nil {
				return "", err
			}
		} else {
			return "", fmt.Errorf("Unknown Google Auth API error: %s, %v", errorDesc, kvs)
		}
	} else {
		masterToken, has = kvs["token"]
		if !has {
			return "", fmt.Errorf("AuhSubToken response does not have master token: %v", kvs)
		}
	}

	log.Debugf("Got master token: %s", masterToken)
	return getSubToken(masterToken)
}

func boolP(value bool) *bool {
	return &value
}

func intP(value int32) *int32 {
	return &value
}

func int64P(value int64) *int64 {
	return &value
}

func stringP(value string) *string {
	return &value
}