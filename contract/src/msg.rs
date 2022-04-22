use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use cosmwasm_std::Binary;

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InstantiateMsg {}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    /// Loop to burn cpu cycles
    CpuLoop { limit: u64 },
    /// Loop making storage calls
    StorageLoop {
        /// Change prefix to increase store size
        prefix: String,
        data: Binary,
        limit: u64,
    },
    /// Allocate large amounts of memory without consuming much gas
    AllocateMemory { pages: u32 },
}
