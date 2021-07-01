package auth

// Based on https://github.com/NoMore201/googleplay-api/blob/master/gpapi/googleplay.py

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/jarijaas/go-gplayapi/pkg/common"
	"github.com/jarijaas/go-gplayapi/pkg/device"
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
	AuthURL    = common.APIBaseURL + "/auth"
	CheckinURL = common.APIBaseURL + "/checkin"
)

/*
Client handles authentication transparently
Decides based on config parameters how authentication should be performed
*/
type Client struct {
	config                 *Config
	deviceConsistencyToken string
}

type Config struct {
	Email                         string
	Password                      string
	GsfId                         string
	AuthSubToken                  string
	DeviceProfile                 *device.Profile
	DeviceCheckinConsistencyToken string
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

	if client.config.GsfId != "" && client.config.AuthSubToken != "" {
		return Token
	}
	return Unknown
}

/*
HasAuthToken check if has necessary tokens (GsfId & AuthSub) for authenticated request, does not check if the tokens are valid
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
	simOperator  string
	roaming      string
}

// Get "androidId", which is a device specific GSF (google services framework) ID
func (client *Client) getGsfId() (string, error) {
	profile := client.config.DeviceProfile

	locale := profile.Locale()
	timezone := profile.Timezone()
	version := int32(3)
	fragment := int32(0)
	id := uint64(0)

	lastCheckinMsec := int64(0)

	log.Debugf("BuildId: %s", profile.BuildId())
	log.Debugf("BuildFingerprint: %s", profile.BuildFingerprint())
	log.Debugf("Locale: %s", locale)
	log.Debugf("Timezone: %s", timezone)
	log.Debugf("Product: %s", profile.BuildProduct())
	log.Debugf("GoogleServices: %d", profile.GoogleServicesVer())
	log.Debugf("Device: %s", profile.ProductDevice())
	log.Debugf("SdkVersion: %d", profile.SdkVer())
	log.Debugf("Model: %s", profile.Model())
	log.Debugf("Manufacturer: %s", profile.Manufacturer())
	log.Debugf("BuildProduct: %s", profile.BuildProduct())
	log.Debugf("GlEsVersion: %d", profile.GlEsVersion())
	log.Debugf("NativePlatforms: %v", profile.NativePlatforms())
	log.Debugf("ScreenWidth: %d", profile.ScreenWidth())
	log.Debugf("ScreenHeigth: %d", profile.ScreenHeight())
	log.Debugf("ScreenDensity: %d", profile.ScreenDensity())

	checkin := pb.AndroidCheckinProto{
		Build: &pb.AndroidBuildProto{
			Id:             stringP(profile.BuildFingerprint()),
			Product:        stringP("unknown"),
			Carrier:        stringP(profile.Brand()),
			Radio:          stringP("unknown"),
			Bootloader:     stringP("unknown"),
			Client:         stringP("unknown"),
			Timestamp:      int64P(time.Now().Unix()),
			GoogleServices: intP(profile.GoogleServicesVer()),
			Device:         stringP(profile.ProductDevice()),
			SdkVersion:     intP(profile.SdkVer()), // the app must support this sdk version
			Model:          stringP(profile.ProductModel()),
			Manufacturer:   stringP(profile.Manufacturer()),
			BuildProduct:   stringP(profile.BuildProduct()),
			OtaInstalled:   boolP(false),
			Clients: []*pb.GclientEntry{
				{
					Id:   intP(2),
					Name: stringP("ms-unknown"),
				},
				{
					Id:   intP(5),
					Name: stringP("mvapp-unknown"),
				},
				{
					Id:   intP(9),
					Name: stringP("ms-unknown"),
				},
				{
					Id:   intP(4),
					Name: stringP("gmm-unknown"),
				},
				{
					Id:   intP(6),
					Name: stringP("am-unknown"),
				},
				{
					Id:   intP(1),
					Name: stringP("unknown"),
				},
			},
		},
		LastCheckinMsec: &lastCheckinMsec,
		Roaming:         stringP("WIFI::"),
		UserNumber:      intP(0),
		ConnectionType:  stringP("WIFI"),
	}

	checkinReq := &pb.AndroidCheckinRequest{
		Imei:          nil,
		Id:            &id,
		Digest:        nil,
		Checkin:       &checkin,
		DesiredBuild:  nil,
		Locale:        &locale,
		LoggingId:     nil,
		MarketCheckin: nil,
		MacAddr:       []string{profile.MacAddr()},
		Meid:          stringP(profile.Meid()),
		AccountCookie: nil,
		TimeZone:      &timezone,
		SecurityToken: nil,
		Version:       &version,
		OtaCert:       profile.OtaCert(),
		SerialNumber:  stringP(profile.SerialNumber()),
		Esn:           nil,
		DeviceConfiguration: &pb.DeviceConfigurationProto{
			TouchScreen:            intP(3),
			Keyboard:               intP(1),
			Navigation:             intP(1),
			ScreenLayout:           intP(2),
			HasHardKeyboard:        boolP(false),
			HasFiveWayNavigation:   boolP(false),
			ScreenDensity:          intP(profile.ScreenDensity()),
			GlEsVersion:            intP(profile.GlEsVersion()),
			SystemSharedLibrary:    strings.Split("android.ext.services,android.ext.shared,android.hidl.manager@1.0-java,android.test.mock,android.test.runner,com.android.future.usb.accessory,com.android.location.provider,com.android.media.remotedisplay,com.android.mediadrm.signer,com.android.nfc_extras,com.dsi.ant.antradio_library,com.google.android.gms,com.google.android.maps,com.google.android.media.effects,com.google.widevine.software.drm,com.qti.dpmapi,com.qti.dpmframework,com.qti.location.sdk,com.qti.snapdragon.sdk.display,com.qualcomm.embmslibrary,com.qualcomm.qcnvitems,com.qualcomm.qcrilhook,com.qualcomm.qti.QtiTelephonyServicelibrary,com.qualcomm.qti.lpa.uimlpalibrary,com.quicinc.cne,com.quicinc.cneapiclient,com.suntek.mway.rcs.client.aidl,com.suntek.mway.rcs.client.api,izat.xt.srv,javax.obex,org.apache.http.legacy,org.lineageos.hardware,org.lineageos.platform", ","),
			NativePlatform:         profile.NativePlatforms(),
			ScreenWidth:            intP(profile.ScreenWidth()),
			ScreenHeight:           intP(profile.ScreenHeight()),
			SystemAvailableFeature: strings.Split("android.hardware.audio.low_latency,android.hardware.audio.output,android.hardware.audio.pro,android.hardware.bluetooth,android.hardware.bluetooth_le,android.hardware.camera,android.hardware.camera.any,android.hardware.camera.autofocus,android.hardware.camera.capability.manual_post_processing,android.hardware.camera.capability.manual_sensor,android.hardware.camera.capability.raw,android.hardware.camera.flash,android.hardware.camera.front,android.hardware.camera.level.full,android.hardware.faketouch,android.hardware.fingerprint,android.hardware.location,android.hardware.location.gps,android.hardware.location.network,android.hardware.microphone,android.hardware.nfc,android.hardware.nfc.any,android.hardware.nfc.hce,android.hardware.opengles.aep,android.hardware.ram.normal,android.hardware.screen.landscape,android.hardware.screen.portrait,android.hardware.sensor.accelerometer,android.hardware.sensor.ambient_temperature,android.hardware.sensor.barometer,android.hardware.sensor.compass,android.hardware.sensor.gyroscope,android.hardware.sensor.hifi_sensors,android.hardware.sensor.light,android.hardware.sensor.proximity,android.hardware.sensor.relative_humidity,android.hardware.sensor.stepcounter,android.hardware.sensor.stepdetector,android.hardware.telephony,android.hardware.telephony.cdma,android.hardware.telephony.gsm,android.hardware.touchscreen,android.hardware.touchscreen.multitouch,android.hardware.touchscreen.multitouch.distinct,android.hardware.touchscreen.multitouch.jazzhand,android.hardware.usb.accessory,android.hardware.usb.host,android.hardware.vr.high_performance,android.hardware.vulkan.compute,android.hardware.vulkan.level,android.hardware.vulkan.version,android.hardware.wifi,android.hardware.wifi.direct,android.software.activities_on_secondary_displays,android.software.app_widgets,android.software.autofill,android.software.backup,android.software.companion_device_setup,android.software.connectionservice,android.software.cts,android.software.device_admin,android.software.home_screen,android.software.input_methods,android.software.live_wallpaper,android.software.managed_users,android.software.midi,android.software.picture_in_picture,android.software.print,android.software.sip,android.software.sip.voip,android.software.voice_recognizers,android.software.vr.mode,android.software.webview,com.google.android.feature.EXCHANGE_6_2,com.google.android.feature.GOOGLE_BUILD,com.google.android.feature.GOOGLE_EXPERIENCE,com.nxp.mifare,org.lineageos.android,org.lineageos.audio,org.lineageos.hardware,org.lineageos.livedisplay,org.lineageos.performance,org.lineageos.profiles,org.lineageos.style,org.lineageos.trust,org.lineageos.weather", ","),
			SystemSupportedLocale:  strings.Split("af,af_ZA,am,am_ET,ar,ar_EG,ar_XB,ast,az,be,bg,bg_BG,bn,bs,ca,ca_ES,cs,cs_CZ,da,da_DK,de,de_AT,de_CH,de_DE,de_LI,el,el_GR,en,en_AU,en_CA,en_GB,en_IN,en_NZ,en_SG,en_US,en_XA,en_XC,eo,es,es_ES,es_US,et,eu,fa,fa_IR,fi,fi_FI,fil,fil_PH,fr,fr_BE,fr_CA,fr_CH,fr_FR,gl,gu,hi,hi_IN,hr,hr_HR,hu,hu_HU,hy,in,in_ID,is,it,it_CH,it_IT,iw,iw_IL,ja,ja_JP,ka,kk,km,kn,ko,ko_KR,ky,lo,lt,lt_LT,lv,lv_LV,mk,ml,mn,mr,ms,ms_MY,my,nb,nb_NO,ne,nl,nl_BE,nl_NL,pa,pl,pl_PL,pt,pt_BR,pt_PT,ro,ro_RO,ru,ru_RU,si,sk,sk_SK,sl,sl_SI,sq,sr,sr_Latn,sr_RS,sv,sv_SE,sw,sw_TZ,ta,te,th,th_TH,tr,tr_TR,uk,uk_UA,ur,uz,vi,vi_VN,zh,zh_CN,zh_HK,zh_TW,zu,zu_ZA", ","),
			GlExtension:            strings.Split("GL_AMD_compressed_ATC_texture,GL_AMD_performance_monitor,GL_ANDROID_extension_pack_es31a,GL_APPLE_texture_2D_limited_npot,GL_ARB_vertex_buffer_object,GL_ARM_shader_framebuffer_fetch_depth_stencil,GL_EXT_EGL_image_array,GL_EXT_YUV_target,GL_EXT_blit_framebuffer_params,GL_EXT_buffer_storage,GL_EXT_clip_cull_distance,GL_EXT_color_buffer_float,GL_EXT_color_buffer_half_float,GL_EXT_copy_image,GL_EXT_debug_label,GL_EXT_debug_marker,GL_EXT_discard_framebuffer,GL_EXT_disjoint_timer_query,GL_EXT_draw_buffers_indexed,GL_EXT_external_buffer,GL_EXT_geometry_shader,GL_EXT_gpu_shader5,GL_EXT_memory_object,GL_EXT_memory_object_fd,GL_EXT_multisampled_render_to_texture,GL_EXT_multisampled_render_to_texture2,GL_EXT_primitive_bounding_box,GL_EXT_protected_textures,GL_EXT_robustness,GL_EXT_sRGB,GL_EXT_sRGB_write_control,GL_EXT_shader_framebuffer_fetch,GL_EXT_shader_io_blocks,GL_EXT_shader_non_constant_global_initializers,GL_EXT_tessellation_shader,GL_EXT_texture_border_clamp,GL_EXT_texture_buffer,GL_EXT_texture_cube_map_array,GL_EXT_texture_filter_anisotropic,GL_EXT_texture_format_BGRA8888,GL_EXT_texture_norm16,GL_EXT_texture_sRGB_R8,GL_EXT_texture_sRGB_decode,GL_EXT_texture_type_2_10_10_10_REV,GL_KHR_blend_equation_advanced,GL_KHR_blend_equation_advanced_coherent,GL_KHR_debug,GL_KHR_no_error,GL_KHR_texture_compression_astc_hdr,GL_KHR_texture_compression_astc_ldr,GL_NV_shader_noperspective_interpolation,GL_OES_EGL_image,GL_OES_EGL_image_external,GL_OES_EGL_image_external_essl3,GL_OES_EGL_sync,GL_OES_blend_equation_separate,GL_OES_blend_func_separate,GL_OES_blend_subtract,GL_OES_compressed_ETC1_RGB8_texture,GL_OES_compressed_paletted_texture,GL_OES_depth24,GL_OES_depth_texture,GL_OES_depth_texture_cube_map,GL_OES_draw_texture,GL_OES_element_index_uint,GL_OES_framebuffer_object,GL_OES_get_program_binary,GL_OES_matrix_palette,GL_OES_packed_depth_stencil,GL_OES_point_size_array,GL_OES_point_sprite,GL_OES_read_format,GL_OES_rgb8_rgba8,GL_OES_sample_shading,GL_OES_sample_variables,GL_OES_shader_image_atomic,GL_OES_shader_multisample_interpolation,GL_OES_standard_derivatives,GL_OES_stencil_wrap,GL_OES_surfaceless_context,GL_OES_texture_3D,GL_OES_texture_compression_astc,GL_OES_texture_cube_map,GL_OES_texture_env_crossbar,GL_OES_texture_float,GL_OES_texture_float_linear,GL_OES_texture_half_float,GL_OES_texture_half_float_linear,GL_OES_texture_mirrored_repeat,GL_OES_texture_npot,GL_OES_texture_stencil8,GL_OES_texture_storage_multisample_2d_array,GL_OES_vertex_array_object,GL_OES_vertex_half_float,GL_OVR_multiview,GL_OVR_multiview2,GL_OVR_multiview_multisampled_render_to_texture,GL_QCOM_alpha_test,GL_QCOM_extended_get,GL_QCOM_shader_framebuffer_fetch_noncoherent,GL_QCOM_texture_foveated,GL_QCOM_tiled_rendering", ","),
			DeviceClass:            nil,
			MaxApkDownloadSizeMb:   nil,
			FeatureFlags: []*pb.FeatureFlag{
				{
					Name:  stringP("android.hardware.audio.output"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.bluetooth"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.bluetooth_le"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.camera"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.camera.any"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.camera.autofocus"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.camera.capability.manual_post_processing"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.camera.capability.manual_sensor"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.camera.capability.raw"),
					Value: boolP(false),
				}, {
					Name:  stringP("android.hardware.camera.flash"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.camera.front"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.camera.level.full"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.faketouch"),
					Value: boolP(false),
				}, {
					Name:  stringP("android.hardware.location"),
					Value: boolP(false),
				}, {
					Name:  stringP("android.hardware.location.gps"),
					Value: boolP(false),
				}, {
					Name:  stringP("android.hardware.location.network"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.microphone"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.nfc"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.nfc.any"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.nfc.hce"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.opengles.aep"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.rm.normal"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.screen.landscape"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.screen.portrait"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.sensor.accelerometer"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.sensor.compass"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.sensor.gyroscope"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.sensor.light"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.sensor.proximity"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.sensor.stepcounter"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.sensor.stepdetector"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.telephony"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.telephony.cdma"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.telephony.gsm"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.touchscreen"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.touchscreen.multitouch"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.touchscreen.multitouch.distinct"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.touchscreen.multitouch.jazzhand"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.usb.accessory"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.usb.host"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.vulkan.compute"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.vulkan.level"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.vulkan.version"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.wifi"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.hardware.wifi.direct"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.activities_on_secondary_displays"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.app_widgets"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.autofill"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.backup"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.cant_save_state"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.companion_device_setup"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.connectionservice"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.cts"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.device_admin"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.home_screen"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.input_methods"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.live_wallpaper"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.managed_users"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.midi"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.picture_in_picture"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.print"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.sip"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.sip.voip"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.verified_boot"),
					Value: boolP(false),
				},
				{
					Name:  stringP("android.software.voice_recognizers"),
					Value: boolP(false),
				}, {
					Name:  stringP("android.software.webview"),
					Value: boolP(false),
				},
				{
					Name:  stringP("com.google.android.feature.TURBO_PRELOAD"),
					Value: boolP(false),
				},
				{
					Name:  stringP("com.nxp.mifare"),
					Value: boolP(false),
				},
				{
					Name:  stringP("com.oneplus.mobilephone"),
					Value: boolP(false),
				},
				{
					Name:  stringP("com.oneplus.software.oos"),
					Value: boolP(false),
				},
				{
					Name:  stringP("com.oneplus.software.oos.n_theme_ready"),
					Value: boolP(false),
				},
				{
					Name:  stringP("com.oneplus.software.oos"),
					Value: boolP(false),
				},
				{
					Name:  stringP("com.oneplus.software.overseas"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.ambient.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.audiotuner.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.authentication_information.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.autobrightctl.animation.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.background.control"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.blackScreenGesture.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.breathLight.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.button.light.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.direct.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.display.soft.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.dualsim.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.finger.print.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.gyroscope.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.hapticsService.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.hw.manufacturer.qualcomm"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.inexact.alarm"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.keyDefine.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.lift_up_display.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.linear.motor.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.op_dark_mode.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.op_dark_mode.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.op_intelligent_background_management.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.op_legal_information.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.opcamera_manual_zsl.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.optical.stabilizer.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.otg.positive.negative.plug.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.otgSwitch.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.picture.color.mode.srgb"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.prox.calibration.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.qcom.fastchager.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.serial_cdev.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.sim_contacts.autosync.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.threeScreenshot.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.threeStageKey.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.timePoweroffWakeup.support"),
					Value: boolP(false),
				},
				{
					Name:  stringP("oem.vooc.fastchager.support"),
					Value: boolP(false),
				},
			},
		},
		MacAddrType:      []string{"wifi"},
		Fragment:         &fragment,
		UserName:         nil,
		UserSerialNumber: intP(0),
	}

	rawMsg, err := proto.Marshal(checkinReq)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", CheckinURL, bytes.NewReader(rawMsg))
	// req, err := http.NewRequest("POST", CheckinURL, bytes.NewReader(testBody))
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/x-protobuf")

	resp, err := http.DefaultClient.Do(req)
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

	/*if len(checkinResp.Setting) == 0 {
		return "", fmt.Errorf("checkin response did not contain settings. the device config was likely invalid")
	}*/

	client.deviceConsistencyToken = checkinResp.GetDeviceCheckinConsistencyToken()
	return strconv.FormatUint(*checkinResp.AndroidId, 16), nil
}

func (client *Client) Authenticate() error {
	var err error

	if client.config.DeviceProfile == nil {
		client.config.DeviceProfile, err = device.LoadDefaultProfile()
		if err != nil {
			return err
		}
	}

	log.Infof("Using device profile: %s", client.config.DeviceProfile.PreferredFilename())

	authType := client.getAuthType()
	if authType == Unknown {
		return fmt.Errorf(
			"could not select authentication type. " +
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

		client.config.AuthSubToken, err = getPlayStoreAuthSubToken(client.config.Email, encryptedPasswd,
			client.config.GsfId, client.config.DeviceProfile)
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
