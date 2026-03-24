#include "wifi.h"

#include "esp_event.h"
#include "esp_log.h"
#include "esp_wifi.h"
#include "freertos/FreeRTOS.h"
#include "freertos/event_groups.h"

#define WIFI_CONNECTED_BIT BIT0
#define WIFI_FAIL_BIT BIT1
#define WIFI_CONNECT_TIMEOUT_MS 30000

static const char *TAG = "wifi";

typedef struct {
  EventGroupHandle_t event_group;
  int retry_num;
} wifi_ctx_t;

static wifi_ctx_t s_wifi_ctx;

static void event_handler(void *arg, esp_event_base_t event_base,
                          int32_t event_id, void *event_data) {
  wifi_ctx_t *ctx = (wifi_ctx_t *)arg;

  if (event_base == WIFI_EVENT && event_id == WIFI_EVENT_STA_START) {
    esp_wifi_connect();
  } else if (event_base == WIFI_EVENT &&
             event_id == WIFI_EVENT_STA_DISCONNECTED) {
    if (ctx->retry_num < CONFIG_ESP_MAXIMUM_RETRY) {
      esp_wifi_connect();
      ctx->retry_num++;
      ESP_LOGI(TAG, "Retrying connection (%d/%d)", ctx->retry_num,
               CONFIG_ESP_MAXIMUM_RETRY);
    } else {
      xEventGroupSetBits(ctx->event_group, WIFI_FAIL_BIT);
      ESP_LOGE(TAG, "Failed to connect after %d retries",
               CONFIG_ESP_MAXIMUM_RETRY);
    }
  } else if (event_base == IP_EVENT && event_id == IP_EVENT_STA_GOT_IP) {
    ip_event_got_ip_t *event = (ip_event_got_ip_t *)event_data;
    ESP_LOGI(TAG, "Connected, IP: " IPSTR, IP2STR(&event->ip_info.ip));
    ctx->retry_num = 0;
    xEventGroupSetBits(ctx->event_group, WIFI_CONNECTED_BIT);
  }
}

esp_err_t wifi_setup(void) {
  s_wifi_ctx.event_group = xEventGroupCreate();
  s_wifi_ctx.retry_num = 0;

  esp_netif_create_default_wifi_sta();

  wifi_init_config_t cfg = WIFI_INIT_CONFIG_DEFAULT();
  esp_err_t ret = esp_wifi_init(&cfg);
  if (ret != ESP_OK) {
    ESP_LOGE(TAG, "esp_wifi_init failed: %s", esp_err_to_name(ret));
    return ret;
  }

  esp_event_handler_instance_t instance_any_id;
  esp_event_handler_instance_t instance_got_ip;
  ret = esp_event_handler_instance_register(WIFI_EVENT, ESP_EVENT_ANY_ID,
                                            &event_handler, &s_wifi_ctx,
                                            &instance_any_id);
  if (ret != ESP_OK) {
    ESP_LOGE(TAG, "Failed to register WIFI_EVENT handler: %s",
             esp_err_to_name(ret));
    return ret;
  }

  ret = esp_event_handler_instance_register(IP_EVENT, IP_EVENT_STA_GOT_IP,
                                            &event_handler, &s_wifi_ctx,
                                            &instance_got_ip);
  if (ret != ESP_OK) {
    ESP_LOGE(TAG, "Failed to register IP_EVENT handler: %s",
             esp_err_to_name(ret));
    return ret;
  }

  wifi_config_t wifi_config = {
      .sta =
          {
              .ssid = CONFIG_ESP_WIFI_SSID,
              .password = CONFIG_ESP_WIFI_PASSWORD,
              .threshold.authmode = WIFI_AUTH_WPA2_PSK,
              .sae_pwe_h2e = WPA3_SAE_PWE_BOTH,
          },
  };
  ret = esp_wifi_set_mode(WIFI_MODE_STA);
  if (ret != ESP_OK) {
    ESP_LOGE(TAG, "esp_wifi_set_mode failed: %s", esp_err_to_name(ret));
    return ret;
  }

  ret = esp_wifi_set_config(WIFI_IF_STA, &wifi_config);
  if (ret != ESP_OK) {
    ESP_LOGE(TAG, "esp_wifi_set_config failed: %s", esp_err_to_name(ret));
    return ret;
  }

  ret = esp_wifi_start();
  if (ret != ESP_OK) {
    ESP_LOGE(TAG, "esp_wifi_start failed: %s", esp_err_to_name(ret));
    return ret;
  }

  ESP_LOGI(TAG, "WiFi STA started, waiting for connection...");

  EventBits_t bits = xEventGroupWaitBits(
      s_wifi_ctx.event_group, WIFI_CONNECTED_BIT | WIFI_FAIL_BIT, pdFALSE,
      pdFALSE, pdMS_TO_TICKS(WIFI_CONNECT_TIMEOUT_MS));

  if (bits & WIFI_CONNECTED_BIT) {
    return ESP_OK;
  }

  if (bits & WIFI_FAIL_BIT) {
    ESP_LOGE(TAG, "Connection failed");
    return ESP_FAIL;
  }

  ESP_LOGE(TAG, "Connection timed out");
  return ESP_ERR_TIMEOUT;
}
