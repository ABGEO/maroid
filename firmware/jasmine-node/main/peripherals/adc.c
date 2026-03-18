#include <esp_adc/adc_oneshot.h>

esp_err_t setup_adc_oneshot_unit(adc_oneshot_unit_handle_t *ret_unit) {
  adc_oneshot_unit_init_cfg_t init_config = {
      .unit_id = ADC_UNIT_1,
      .ulp_mode = ADC_ULP_MODE_DISABLE,
  };

  return adc_oneshot_new_unit(&init_config, ret_unit);
}
