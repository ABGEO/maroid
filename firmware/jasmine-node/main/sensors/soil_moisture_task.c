#include "soil_moisture_task.h"
#include "mqtt.h"
#include "soil_moisture.h"

#include "esp_log.h"
#include "freertos/event_groups.h"
#include "freertos/task.h"

#define SENSOR_READING_BUF_SIZE 32

static const char *TAG = "soil_moisture_task";

void soil_moisture_task(void *arg) {
  soil_moisture_task_params_t *params = (soil_moisture_task_params_t *)arg;

  soil_moisture_t sm_ctx;
  esp_err_t err = soil_moisture_setup(&sm_ctx, params->adc_handle,
                                      CONFIG_SOIL_MOISTURE_ADC1_CH);
  if (err != ESP_OK) {
    ESP_LOGE(TAG, "soil_moisture_setup failed: %s", esp_err_to_name(err));
    xEventGroupSetBits(params->event_group,
                       SOIL_MOISTURE_DONE_BIT | SOIL_MOISTURE_ERR_BIT);
    vTaskDelete(NULL);
    return;
  }

  uint32_t reading = soil_moisture_read(&sm_ctx);
  char reading_buf[SENSOR_READING_BUF_SIZE];

  if (reading > CONFIG_SOIL_MOISTURE_MAX_THRESHOLD ||
      reading < CONFIG_SOIL_MOISTURE_MIN_THRESHOLD) {
    ESP_LOGE(TAG, "Invalid Threshold: %lu", reading);
    xEventGroupSetBits(params->event_group,
                       SOIL_MOISTURE_DONE_BIT | SOIL_MOISTURE_ERR_BIT);
    vTaskDelete(NULL);
    return;
  }

  float soil_moisture_normalized =
      soil_moisture_normalize(CONFIG_SOIL_MOISTURE_MIN_THRESHOLD,
                              CONFIG_SOIL_MOISTURE_MAX_THRESHOLD, reading);
  ESP_LOGI(TAG, "Soil Moisture: %f%% | Raw: %lu", soil_moisture_normalized,
           reading);

  snprintf(reading_buf, sizeof(reading_buf), "%f", soil_moisture_normalized);
  mqtt_send_reading(params->mqtt_client, "soil-moisture", reading_buf);

  snprintf(reading_buf, sizeof(reading_buf), "%lu", reading);
  mqtt_send_reading(params->mqtt_client, "soil-moisture-raw", reading_buf);

  xEventGroupSetBits(params->event_group, SOIL_MOISTURE_DONE_BIT);
  vTaskDelete(NULL);
}
