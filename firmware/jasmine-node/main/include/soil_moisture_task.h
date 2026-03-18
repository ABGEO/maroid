#pragma once

#include "adc.h"
#include "freertos/FreeRTOS.h"

#include <mqtt_client.h>

#define SOIL_MOISTURE_DONE_BIT BIT2
#define SOIL_MOISTURE_ERR_BIT BIT6

typedef struct {
  adc_oneshot_unit_handle_t adc_handle;
  esp_mqtt_client_handle_t mqtt_client;
  EventGroupHandle_t event_group;
} soil_moisture_task_params_t;

void soil_moisture_task(void *arg);
