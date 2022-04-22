use cosmwasm_std::{entry_point, Binary, DepsMut, Env, MessageInfo, Response, StdError};

use crate::errors::ContractError;
use crate::msg::{ExecuteMsg, InstantiateMsg};

#[entry_point]
pub fn instantiate(
    _deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    _msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    Ok(Response::default())
}

#[entry_point]
pub fn execute(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::CpuLoop { limit } => do_cpu_loop(limit),
        ExecuteMsg::StorageLoop {
            prefix,
            data,
            limit,
        } => do_storage_loop(deps, prefix, data, limit),
        ExecuteMsg::AllocateMemory { pages } => do_allocate_large_memory(pages),
    }
}

#[entry_point]
pub fn migrate(_deps: DepsMut, _env: Env, _msg: InstantiateMsg) -> Result<Response, ContractError> {
    Ok(Response::default())
}

fn do_cpu_loop(limit: u64) -> Result<Response, ContractError> {
    let mut counter = 0u64;
    loop {
        loop_cycle(1_000_000_000_000_000);

        if counter >= limit {
            break;
        }
        counter += 1;
    }

    Ok(Response::default().add_attribute("action", "cpu_loop"))
}

fn loop_cycle(size: u64) {
    let mut counter = 0u64;
    loop {
        counter += 1;
        if counter >= size {
            break;
        }
    }
}

fn do_storage_loop(
    deps: DepsMut,
    prefix: String,
    data: Binary,
    limit: u64,
) -> Result<Response, ContractError> {
    let mut counter = 0u64;
    loop {
        if counter >= limit {
            break;
        }

        let key = format!("{}_{}", prefix, counter);
        deps.storage.set(key.as_bytes(), data.as_slice());
        counter += 1;
    }

    Ok(Response::default().add_attribute("action", "storage_loop"))
}

#[allow(unused_variables)]
fn do_allocate_large_memory(pages: u32) -> Result<Response, ContractError> {
    // We create memory pages explicitely since Rust's default allocator seems to be clever enough
    // to not grow memory for unused capacity like `Vec::<u8>::with_capacity(100 * 1024 * 1024)`.
    // Even with std::alloc::alloc the memory did now grow beyond 1.5 MiB.

    #[cfg(target_arch = "wasm32")]
    {
        use core::arch::wasm32;
        let old_size = wasm32::memory_grow(0, pages as usize);
        if old_size == usize::max_value() {
            return Err(StdError::generic_err("memory.grow failed").into());
        }
        Ok(Response::new().set_data((old_size as u32).to_be_bytes()))
    }

    #[cfg(not(target_arch = "wasm32"))]
    Err(StdError::generic_err("Unsupported architecture").into())
}

#[cfg(test)]
mod tests {
    use super::*;
    use cosmwasm_std::testing::{mock_dependencies, mock_env, mock_info};

    #[test]
    fn proper_initialization() {
        let mut deps = mock_dependencies();
        let creator = String::from("creator");

        let msg = InstantiateMsg {};
        let info = mock_info(creator.as_str(), &[]);
        let res = instantiate(deps.as_mut(), mock_env(), info, msg).unwrap();
        assert_eq!(res.messages.len(), 0);
    }
}
