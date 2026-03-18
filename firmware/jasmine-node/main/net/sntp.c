#include "sntp.h"

#include "esp_log.h"
#include "esp_netif_sntp.h"
#include "event_wait.h"
#include "freertos/FreeRTOS.h"
#include "freertos/event_groups.h"

#define SNTP_SYNCED_BIT BIT0
#define SNTP_SYNC_TIMEOUT_MS 30000

static const char *TAG = "sntp";

static EventGroupHandle_t s_sntp_event_group;

static void sntp_sync_cb(struct timeval *tv) {
  xEventGroupSetBits(s_sntp_event_group, SNTP_SYNCED_BIT);
}

esp_err_t sntp_setup(void) {
  s_sntp_event_group = xEventGroupCreate();

  ESP_LOGI(TAG, "Initializing and starting SNTP");

  esp_sntp_config_t config =
      ESP_NETIF_SNTP_DEFAULT_CONFIG(CONFIG_SNTP_TIME_SERVER);
  config.sync_cb = sntp_sync_cb;

  esp_err_t err = esp_netif_sntp_init(&config);
  if (err != ESP_OK) {
    vEventGroupDelete(s_sntp_event_group);

    return err;
  }

  esp_err_t wait_err = event_wait(s_sntp_event_group, SNTP_SYNCED_BIT,
                                  pdMS_TO_TICKS(SNTP_SYNC_TIMEOUT_MS), TAG);
  vEventGroupDelete(s_sntp_event_group);

  if (wait_err != ESP_OK) {
    esp_netif_sntp_deinit();

    return wait_err;
  }

  ESP_LOGI(TAG, "SNTP initialized");
  return ESP_OK;
}
