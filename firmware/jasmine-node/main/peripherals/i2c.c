#include "i2c.h"

esp_err_t i2c_master_bus_init(i2c_master_bus_handle_t *ret_bus) {
  i2c_master_bus_config_t bus_config = {
      .i2c_port = I2C_NUM_0,
      .sda_io_num = CONFIG_I2C_MASTER_SDA_IO,
      .scl_io_num = CONFIG_I2C_MASTER_SCL_IO,
      .clk_source = I2C_CLK_SRC_DEFAULT,
      .glitch_ignore_cnt = 7,
      .flags.enable_internal_pullup = true,
  };

  return i2c_new_master_bus(&bus_config, ret_bus);
}
