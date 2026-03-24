#pragma once

#include "esp_err.h"
#include <esp_adc/adc_oneshot.h>

esp_err_t adc_oneshot_unit_init(adc_oneshot_unit_handle_t *ret_unit);
