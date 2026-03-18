#pragma once

#include "driver/i2c_master.h"
#include "freertos/FreeRTOS.h"
#include "freertos/event_groups.h"

#include <mqtt_client.h>

#define BH1750_DONE_BIT BIT0
#define BH1750_ERR_BIT BIT4

typedef struct {
  i2c_master_bus_handle_t bus;
  esp_mqtt_client_handle_t mqtt_client;
  EventGroupHandle_t event_group;
} bh1750_task_params_t;

void bh1750_task(void *arg);
