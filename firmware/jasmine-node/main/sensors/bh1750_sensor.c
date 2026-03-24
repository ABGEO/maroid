#include "bh1750_sensor.h"
#include "mqtt.h"

#include "bh1750.h"
#include "esp_log.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"

#define BH1750_MEASUREMENT_DELAY_MS 180
#define READING_BUF_SIZE 32

static const char *TAG = "bh1750";

static bh1750_handle_t s_handle;

esp_err_t bh1750_sensor_init(void *ctx) {
  bh1750_ctx_t *bh = (bh1750_ctx_t *)ctx;

  esp_err_t err = bh1750_create(bh->i2c_bus, BH1750_I2C_ADDRESS_DEFAULT,
                                &s_handle);
  if (err != ESP_OK) {
    ESP_LOGE(TAG, "bh1750_create failed: %s", esp_err_to_name(err));
    return err;
  }

  err = bh1750_power_on(s_handle);
  if (err != ESP_OK) {
    ESP_LOGE(TAG, "bh1750_power_on failed: %s", esp_err_to_name(err));
    return err;
  }

  err = bh1750_set_measure_mode(s_handle, BH1750_ONETIME_1LX_RES);
  if (err != ESP_OK) {
    ESP_LOGE(TAG, "bh1750_set_measure_mode failed: %s", esp_err_to_name(err));
    return err;
  }

  vTaskDelay(pdMS_TO_TICKS(BH1750_MEASUREMENT_DELAY_MS));
  return ESP_OK;
}

esp_err_t bh1750_sensor_read(void *ctx, esp_mqtt_client_handle_t client) {
  float reading;
  esp_err_t err = bh1750_get_data(s_handle, &reading);
  if (err != ESP_OK) {
    ESP_LOGE(TAG, "bh1750_get_data failed: %s", esp_err_to_name(err));
    return err;
  }

  ESP_LOGI(TAG, "Illuminance: %.1f lux", reading);

  char buf[READING_BUF_SIZE];
  snprintf(buf, sizeof(buf), "%.2f", reading);
  int msg_id = mqtt_send_reading(client, "illuminance", buf);
  return msg_id >= 0 ? ESP_OK : ESP_FAIL;
}

void bh1750_sensor_cleanup(void *ctx) {
  if (s_handle != NULL) {
    bh1750_delete(s_handle);
    s_handle = NULL;
  }
}
