#include "sensor_task.h"
#include "sensor_registry.h"

#include "esp_log.h"
#include "freertos/FreeRTOS.h"
#include "freertos/event_groups.h"
#include "freertos/task.h"

#define SENSOR_MAX_COUNT 12
#define SENSOR_WAIT_TIMEOUT_MS 30000

static const char *TAG = "sensor_task";

typedef struct {
    const sensor_descriptor_t *descriptor;
    esp_mqtt_client_handle_t mqtt_client;
    EventGroupHandle_t event_group;
    size_t index;
    size_t total;
} sensor_task_params_t;

static void sensor_task(void *arg) {
  sensor_task_params_t *params = (sensor_task_params_t *)arg;
  const sensor_descriptor_t *desc = params->descriptor;
  EventBits_t done_bit = BIT(params->index);
  EventBits_t err_bit = BIT(params->index + params->total);

  esp_err_t err = ESP_OK;

  if (desc->init != NULL) {
    err = desc->init(desc->ctx);
    if (err != ESP_OK) {
      ESP_LOGE(TAG, "%s init failed: %s", desc->name, esp_err_to_name(err));
    }
  }

  if (err == ESP_OK) {
    err = desc->read(desc->ctx, params->mqtt_client);
    if (err != ESP_OK) {
      ESP_LOGE(TAG, "%s read failed: %s", desc->name, esp_err_to_name(err));
    }
  }

  if (desc->cleanup != NULL) {
    desc->cleanup(desc->ctx);
  }

  EventBits_t bits = done_bit;
  if (err != ESP_OK) {
    bits |= err_bit;
  }
  xEventGroupSetBits(params->event_group, bits);
  vTaskDelete(NULL);
}

esp_err_t sensors_run(esp_mqtt_client_handle_t mqtt_client) {
  size_t count = sensor_count();
  if (count == 0) {
    ESP_LOGW(TAG, "No sensors enabled");
    return ESP_OK;
  }

  const sensor_descriptor_t *sensors = sensor_list();

  EventGroupHandle_t event_group = xEventGroupCreate();
  if (event_group == NULL) {
    ESP_LOGE(TAG, "Failed to create event group");
    return ESP_ERR_NO_MEM;
  }

  static sensor_task_params_t params[SENSOR_MAX_COUNT];
  EventBits_t all_done_bits = 0;

  for (size_t i = 0; i < count; i++) {
    params[i] = (sensor_task_params_t){
        .descriptor = &sensors[i],
        .mqtt_client = mqtt_client,
        .event_group = event_group,
        .index = i,
        .total = count,
    };

    if (xTaskCreate(sensor_task, sensors[i].name, sensors[i].stack_size,
                    &params[i], 5, NULL) != pdPASS) {
      ESP_LOGE(TAG, "Failed to create task for %s", sensors[i].name);
      xEventGroupSetBits(event_group, BIT(i) | BIT(i + count));
    }

    all_done_bits |= BIT(i);
  }

  EventBits_t done = xEventGroupWaitBits(event_group, all_done_bits, pdFALSE,
                                         pdTRUE,
                                         pdMS_TO_TICKS(SENSOR_WAIT_TIMEOUT_MS));

  bool all_completed = (done & all_done_bits) == all_done_bits;

  for (size_t i = 0; i < count; i++) {
    if (done & BIT(i + count)) {
      ESP_LOGW(TAG, "Sensor '%s' reported an error", sensors[i].name);
    } else if (!(done & BIT(i))) {
      ESP_LOGW(TAG, "Sensor '%s' timed out", sensors[i].name);
    }
  }

  if (all_completed) {
    vEventGroupDelete(event_group);
  } else {
    ESP_LOGW(TAG, "Some sensors timed out, leaking event group to avoid UB");
  }

  return ESP_OK;
}
