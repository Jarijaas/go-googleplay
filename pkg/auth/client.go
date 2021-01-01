package auth

// Based on https://github.com/NoMore201/googleplay-api/blob/master/gpapi/googleplay.py

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/jarijaas/go-gplayapi/pkg/common"
	"github.com/jarijaas/go-gplayapi/pkg/keyring"
	"github.com/jarijaas/go-gplayapi/pkg/playstore/pb"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	AuthURL = common.APIBaseURL + "/auth"
	CheckinURL = common.APIBaseURL + "/checkin"
)

/**
Handles authentication transparently
Decides based on config parameters how authentication should be performed
 */
type Client struct {
	config *Config
	deviceConsistencyToken string
}

type Config struct {
	Email string
	Password string
	GsfId string
	AuthSubToken string
}

func CreatePlaystoreAuthClient(config *Config) (*Client, error) {
	if config.GsfId == "" && config.AuthSubToken == "" {
		gsfId, authSub, err := keyring.GetGoogleTokens()
		if err == nil && gsfId != "" && authSub != "" {
			log.Tracef("Found GSIF %s and authSub %s tokens from keyring", gsfId, authSub)
			config.GsfId = gsfId
			config.AuthSubToken = authSub
		}
	}
	return &Client{config: config}, nil
}

type Type string

const (
	EmailPassword Type = "email-pass"
	Token         Type = "token"
	Unknown       Type = ""
)

/**
Use email and passwd if set, otherwise use tokens
 */
func (client *Client) getAuthType() Type {
	if client.config.Email != "" && client.config.Password != "" {
		return EmailPassword
	}

	if client.config.GsfId != ""  && client.config.AuthSubToken  != "" {
		return Token
	}
	return Unknown
}

/**
Check if has necessary tokens (GsfId & AuthSub) for authenticated request, does not check if the tokens are valid
 */
func (client *Client) HasAuthToken() bool {
	return client.config.GsfId != "" && client.config.AuthSubToken != ""
}

func (client *Client) GetGsfId() string {
	return client.config.GsfId
}

func (client *Client) GetAuthSubToken() string {
	return client.config.AuthSubToken
}

type DeviceConfig struct {
	cellOperator string
	simOperator string
	roaming string
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

// Get "androidId", which is a device specific GSF (google services framework) ID
func (client *Client) getGsfId() (string, error) {
	username := "username"

	locale := "fi"
	timezone := "Europe/Helsinki"
	version := int32(3)
	fragment := int32(0)

	lastCheckinMsec := int64(0)
	userNumber := int32(0)
	cellOperator := "22210"
	simOperator := "22210"
	roaming := "mobile-notroaming"

	checkin := pb.AndroidCheckinProto{
		Build:           &pb.AndroidBuildProto{
			Id:             stringP("Jolla/alien_jolla_bionic/alien_jolla_bionic:4.1.2/JZO54K/eng.erin.20151105.144423:user/dev-keys"),
			Product:        stringP("unknown"),
			Carrier:        stringP("Jolla"),
			Radio:          stringP("unknown"),
			Bootloader:     stringP("unknown"),
			Client:         stringP("android-google"),
			Timestamp:      int64P(int64(time.Now().Second())),
			GoogleServices: intP(12673002),
			Device:         stringP("alien_jolla_bionic"),
			SdkVersion:     intP(16),
			Model:          stringP("Jolla"),
			Manufacturer:   stringP("unknown"),
			BuildProduct:   stringP("alien_jolla_bionic"),
			OtaInstalled:   boolP(true),
		},
		LastCheckinMsec: &lastCheckinMsec,
		Event:           nil,
		Stat:            nil,
		RequestedGroup:  nil,
		CellOperator:    &cellOperator,
		SimOperator:     &simOperator,
		Roaming:         &roaming,
		UserNumber:      &userNumber,
	}

	checkinReq := &pb.AndroidCheckinRequest{
		Imei:                nil,
		Id:                  nil,
		Digest:              nil,
		Checkin:             &checkin,
		DesiredBuild:        nil,
		Locale:              &locale,
		LoggingId:           nil,
		MarketCheckin:       nil,
		MacAddr:             nil,
		Meid:                nil,
		AccountCookie:       nil,
		TimeZone:            &timezone,
		SecurityToken:       nil,
		Version:             &version,
		OtaCert:             nil,
		SerialNumber:        nil,
		Esn:                 nil,
		DeviceConfiguration: &pb.DeviceConfigurationProto{
			TouchScreen:            intP(3),
			Keyboard:               intP(2),
			Navigation:             intP(2),
			ScreenLayout:           intP(2),
			HasHardKeyboard:        boolP(true),
			HasFiveWayNavigation:   boolP(true),
			ScreenDensity:          intP(240),
			GlEsVersion:            intP(131072),
			SystemSharedLibrary:    strings.Split("android.test.runner,com.android.location.provider,com.google.android.maps,com.google.android.media.effects,com.google.widevine.software.drm,javax.btobex,javax.obex", ","),
			SystemAvailableFeature: strings.Split("android.hardware.camera,android.hardware.camera.autofocus,android.hardware.location,android.hardware.location.gps,android.hardware.location.network,android.hardware.microphone,android.hardware.screen.landscape,android.hardware.screen.portrait,android.hardware.sensor.accelerometer,android.hardware.sensor.compass,android.hardware.sensor.light,android.hardware.sensor.proximity,android.hardware.telephony,android.hardware.telephony.gsm,android.hardware.touchscreen,android.hardware.touchscreen.multitouch,android.hardware.touchscreen.multitouch.jazzhand,android.hardware.wifi,com.google.android.feature.GOOGLE_BUILD,com.myriadgroup.alien", ","),
			NativePlatform:         []string{"armeabi-v7a", "armeabi"},
			ScreenWidth:            intP(540),
			ScreenHeight:           intP(888),
			SystemSupportedLocale:  []string{"fi_FI"},
			GlExtension:            strings.Split("GL_AMD_compressed_ATC_texture,GL_AMD_performance_monitor,GL_ANDROID_extension_pack_es31a,GL_APPLE_texture_2D_limited_npot,GL_ARB_vertex_buffer_object,GL_EXT_EGL_image_array,GL_EXT_YUV_target,GL_EXT_blit_framebuffer_params,GL_EXT_buffer_storage,GL_EXT_color_buffer_float,GL_EXT_color_buffer_half_float,GL_EXT_copy_image,GL_EXT_debug_label,GL_EXT_debug_marker,GL_EXT_discard_framebuffer,GL_EXT_disjoint_timer_query,GL_EXT_draw_buffers_indexed,GL_EXT_external_buffer,GL_EXT_geometry_shader,GL_EXT_gpu_shader5,GL_EXT_multisampled_render_to_texture,GL_EXT_multisampled_render_to_texture2,GL_EXT_primitive_bounding_box,GL_EXT_protected_textures,GL_EXT_robustness,GL_EXT_sRGB,GL_EXT_sRGB_write_control,GL_EXT_shader_io_blocks,GL_EXT_shader_non_constant_global_initializers,GL_EXT_tessellation_shader,GL_EXT_texture_border_clamp,GL_EXT_texture_buffer,GL_EXT_texture_cube_map_array,GL_EXT_texture_filter_anisotropic,GL_EXT_texture_format_BGRA8888,GL_EXT_texture_norm16,GL_EXT_texture_sRGB_R8,GL_EXT_texture_sRGB_decode,GL_EXT_texture_type_2_10_10_10_REV,GL_KHR_blend_equation_advanced,GL_KHR_blend_equation_advanced_coherent,GL_KHR_debug,GL_KHR_no_error,GL_KHR_texture_compression_astc_ldr,GL_NV_shader_noperspective_interpolation,GL_OES_EGL_image,GL_OES_EGL_image_external,GL_OES_EGL_image_external_essl3,GL_OES_EGL_sync,GL_OES_blend_equation_separate,GL_OES_blend_func_separate,GL_OES_blend_subtract,GL_OES_compressed_ETC1_RGB8_texture,GL_OES_compressed_paletted_texture,GL_OES_depth24,GL_OES_depth_texture,GL_OES_depth_texture_cube_map,GL_OES_draw_texture,GL_OES_element_index_uint,GL_OES_framebuffer_object,GL_OES_get_program_binary,GL_OES_matrix_palette,GL_OES_packed_depth_stencil,GL_OES_point_size_array,GL_OES_point_sprite,GL_OES_read_format,GL_OES_rgb8_rgba8,GL_OES_sample_shading,GL_OES_sample_variables,GL_OES_shader_image_atomic,GL_OES_shader_multisample_interpolation,GL_OES_standard_derivatives,GL_OES_stencil_wrap,GL_OES_surfaceless_context,GL_OES_texture_3D,GL_OES_texture_cube_map,GL_OES_texture_env_crossbar,GL_OES_texture_float,GL_OES_texture_float_linear,GL_OES_texture_half_float,GL_OES_texture_half_float_linear,GL_OES_texture_mirrored_repeat,GL_OES_texture_npot,GL_OES_texture_stencil8,GL_OES_texture_storage_multisample_2d_array,GL_OES_vertex_array_object,GL_OES_vertex_half_float,GL_OVR_multiview,GL_OVR_multiview2,GL_OVR_multiview_multisampled_render_to_texture,GL_QCOM_alpha_test,GL_QCOM_extended_get,GL_QCOM_shader_framebuffer_fetch_noncoherent,GL_QCOM_tiled_rendering", ","),
			DeviceClass:            nil,
			MaxApkDownloadSizeMb:   intP(10 * 100),
		},
		MacAddrType:         nil,
		Fragment:            &fragment,
		UserName:            &username,
		UserSerialNumber:    nil,
	}


	rawMsg, err := proto.Marshal(checkinReq)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(CheckinURL, "application/x-protobuf", bytes.NewReader(rawMsg))
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var checkinResp pb.AndroidCheckinResponse
	err = proto.Unmarshal(body, &checkinResp)
	if err != nil {
		return "", err
	}

	rawMsg, err = proto.Marshal(checkinReq)
	if err != nil {
		return "", err
	}

	resp, err = http.Post(CheckinURL, "application/x-protobuf", bytes.NewReader(rawMsg))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("checkin error: %s %d", resp.Status, resp.StatusCode)
	}

	return strconv.FormatUint(*checkinResp.AndroidId, 16), nil
}


func (client *Client) Authenticate() error {
	log.Debugf("Authenticate")

	authType := client.getAuthType()
	if authType == Unknown {
		return fmt.Errorf(
			"could not select authentication type." +
				"Did you specify the email and the password or alternatively GSFID and authSubToken")
	}

	switch authType {
	case EmailPassword:
		encryptedPasswd, err := encryptCredentials(client.config.Email, client.config.Password, nil)
		if err != nil {
			return err
		}

		client.config.GsfId, err = client.getGsfId()
		if err != nil {
			return err
		}

		client.config.AuthSubToken, err = getPlayStoreAuthSubToken(client.config.Email, encryptedPasswd)
		if err != nil {
			return err
		}

		log.Infof("Got GsfId and AuthSubToken, saving these to keyring")

		err = keyring.SaveToken(keyring.GSFID, client.config.GsfId)
		if err != nil {
			return err
		}

		err = keyring.SaveToken(keyring.AuthSubToken, client.config.AuthSubToken)
		if err != nil {
			return err
		}
	}

	return nil
}
