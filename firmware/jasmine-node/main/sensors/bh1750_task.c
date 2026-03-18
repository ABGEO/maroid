#include "bh1750_task.h"
#include "mqtt.h"

#include "bh1750.h"
#include "esp_check.h"
#include "esp_log.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"

/* BH1750 datasheet: H-Resolution mode max measurement time is 120 ms.
 * 180 ms adds a conservative margin before reading. */
#define BH1750_MEASUREMENT_DELAY_MS 180

#define SENSOR_READING_BUF_SIZE 32

static const char *TAG = "bh1750_task";

static void bh1750_cleanup(bh1750_handle_t *handle,
                           EventGroupHandle_t event_group, bool error) {
  if (handle != NULL) {
    bh1750_delete(handle);
  }

  EventBits_t bits = BH1750_DONE_BIT;
  if (error) {
    bits |= BH1750_ERR_BIT;
  }
  xEventGroupSetBits(event_group, bits);
  vTaskDelete(NULL);
}

static esp_err_t bh1750_init(i2c_master_bus_handle_t i2c_bus,
                             bh1750_handle_t *handle_ret) {
  ESP_RETURN_ON_ERROR(
      bh1750_create(i2c_bus, BH1750_I2C_ADDRESS_DEFAULT, handle_ret), TAG,
      "bh1750_create failed");
  ESP_RETURN_ON_ERROR(bh1750_power_on(*handle_ret), TAG,
                      "bh1750_power_on failed");
  ESP_RETURN_ON_ERROR(
      bh1750_set_measure_mode(*handle_ret, BH1750_ONETIME_1LX_RES), TAG,
      "bh1750_set_measure_mode failed");
  return ESP_OK;
}

void bh1750_task(void *arg) {
  bh1750_task_params_t *params = (bh1750_task_params_t *)arg;
  bh1750_handle_t handle = NULL;
  float reading;
  char reading_buf[SENSOR_READING_BUF_SIZE];

  esp_err_t err = bh1750_init(params->bus, &handle);
  if (err != ESP_OK) {
    bh1750_cleanup(handle, params->event_group, true);
    return;
  }

  vTaskDelay(pdMS_TO_TICKS(BH1750_MEASUREMENT_DELAY_MS));

  if (bh1750_get_data(handle, &reading) == ESP_OK) {
    ESP_LOGI(TAG, "Illuminance: %.1f lux", reading);

    snprintf(reading_buf, sizeof(reading_buf), "%f", reading);
    mqtt_send_reading(params->mqtt_client, "illuminance", reading_buf);
    bh1750_cleanup(handle, params->event_group, false);
  } else {
    ESP_LOGE(TAG, "Failed to read sensor");
    bh1750_cleanup(handle, params->event_group, true);
  }
}
