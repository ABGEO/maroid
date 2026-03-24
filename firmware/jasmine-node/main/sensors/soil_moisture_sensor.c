#include "soil_moisture_sensor.h"
#include "mqtt.h"
#include "soil_moisture.h"

#include "esp_log.h"

#define READING_BUF_SIZE 32

static const char *TAG = "soil_moisture";

static soil_moisture_t s_sm_ctx;

esp_err_t soil_moisture_sensor_init(void *ctx) {
  soil_moisture_ctx_t *sm = (soil_moisture_ctx_t *)ctx;

  soil_moisture_config_t config = {
      .sample_count = CONFIG_SOIL_MOISTURE_SAMPLE_COUNT,
      .sample_delay_ms = CONFIG_SOIL_MOISTURE_SAMPLE_DELAY_MS,
      .valid_min = CONFIG_SOIL_MOISTURE_VALID_MIN,
      .valid_max = CONFIG_SOIL_MOISTURE_VALID_MAX,
      .cal_air_value = CONFIG_SOIL_MOISTURE_CAL_AIR_VALUE,
      .cal_water_value = CONFIG_SOIL_MOISTURE_CAL_WATER_VALUE,
  };

  return soil_moisture_setup(&s_sm_ctx, sm->adc_handle,
                             CONFIG_SOIL_MOISTURE_ADC1_CH, &config);
}

esp_err_t soil_moisture_sensor_read(void *ctx,
                                    esp_mqtt_client_handle_t client) {
  uint32_t reading;
  esp_err_t err = soil_moisture_read(&s_sm_ctx, &reading);
  if (err != ESP_OK) {
    ESP_LOGE(TAG, "soil_moisture_read failed: %s", esp_err_to_name(err));
    return err;
  }

  float normalized = soil_moisture_normalize(&s_sm_ctx, reading);
  ESP_LOGI(TAG, "Soil Moisture: %.2f%% | Raw: %lu", normalized,
           (unsigned long)reading);

  char buf[READING_BUF_SIZE];
  int msg_id;

  snprintf(buf, sizeof(buf), "%.2f", normalized);
  msg_id = mqtt_send_reading(client, "soil-moisture", buf);
  if (msg_id < 0) {
    return ESP_FAIL;
  }

  snprintf(buf, sizeof(buf), "%lu", (unsigned long)reading);
  msg_id = mqtt_send_reading(client, "soil-moisture-raw", buf);
  return msg_id >= 0 ? ESP_OK : ESP_FAIL;
}
