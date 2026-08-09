use std::io::{self, Read};
use std::path::{Path, PathBuf};
use std::process::ExitCode;

use libloot::{Game, GameType};
use serde::{Deserialize, Serialize};

const CONTRACT: &str = "dmm-loot-sorter.v1";

#[derive(Debug, Deserialize)]
struct SortRequest {
    game_id: String,
    masterlist: String,
    #[serde(default)]
    userlist: String,
    #[serde(default)]
    prelude: String,
    game_path: String,
    #[serde(default)]
    game_local_path: String,
    plugins: Vec<SortPlugin>,
    #[serde(default)]
    current_order: Vec<String>,
    contract: String,
}

#[derive(Debug, Deserialize)]
struct SortPlugin {
    name: String,
    path: String,
}

#[derive(Debug, Serialize)]
struct SortResponse {
    sorted_plugins: Vec<String>,
    warnings: Vec<String>,
    engine: &'static str,
}

fn main() -> ExitCode {
    match run() {
        Ok(()) => ExitCode::SUCCESS,
        Err(err) => {
            eprintln!("{err}");
            ExitCode::FAILURE
        }
    }
}

fn run() -> Result<(), Box<dyn std::error::Error>> {
    let mut body = String::new();
    io::stdin().read_to_string(&mut body)?;
    let request: SortRequest = serde_json::from_str(&body)?;
    validate_request(&request)?;

    let game_type = game_type_for_id(&request.game_id)
        .ok_or_else(|| format!("unsupported LOOT game id {:?}", request.game_id))?;
    let game_path = PathBuf::from(request.game_path.trim());
    let local_path = request.game_local_path.trim();
    let mut game = if local_path.is_empty() {
        Game::new(game_type, &game_path)?
    } else {
        std::fs::create_dir_all(local_path)?;
        Game::with_local_path(game_type, &game_path, Path::new(local_path))?
    };

    load_lists(&game, &request)?;
    game.load_current_load_order_state()?;

    let plugin_paths = plugin_paths(&request)?;
    let plugin_refs: Vec<&Path> = plugin_paths.iter().map(PathBuf::as_path).collect();
    game.load_plugins(&plugin_refs)?;

    let order = current_order(&request);
    let order_refs: Vec<&str> = order.iter().map(String::as_str).collect();
    let sorted = game.sort_plugins(&order_refs)?;
    serde_json::to_writer(
        io::stdout(),
        &SortResponse {
            sorted_plugins: sorted,
            warnings: Vec::new(),
            engine: "libloot",
        },
    )?;
    Ok(())
}

fn validate_request(request: &SortRequest) -> Result<(), String> {
    if request.contract != CONTRACT {
        return Err(format!(
            "unsupported sorter contract {:?}; expected {CONTRACT}",
            request.contract
        ));
    }
    if request.game_id.trim().is_empty() {
        return Err("game_id is required".to_owned());
    }
    if request.game_path.trim().is_empty() {
        return Err("game_path is required".to_owned());
    }
    if request.masterlist.trim().is_empty() {
        return Err("masterlist is required".to_owned());
    }
    if request.plugins.is_empty() {
        return Err("at least one plugin is required".to_owned());
    }
    for plugin in &request.plugins {
        if plugin.name.trim().is_empty() {
            return Err("plugin name is required".to_owned());
        }
        if plugin.path.trim().is_empty() {
            return Err(format!("plugin {:?} path is required", plugin.name));
        }
    }
    Ok(())
}

fn load_lists(game: &Game, request: &SortRequest) -> Result<(), Box<dyn std::error::Error>> {
    let database = game.database();
    let mut database = database
        .write()
        .map_err(|_| "libloot database lock was poisoned")?;
    let masterlist = Path::new(request.masterlist.trim());
    let prelude = request.prelude.trim();
    if prelude.is_empty() {
        database.load_masterlist(masterlist)?;
    } else {
        database.load_masterlist_with_prelude(masterlist, Path::new(prelude))?;
    }

    let userlist = request.userlist.trim();
    if !userlist.is_empty() {
        let userlist_path = Path::new(userlist);
        if userlist_path.is_file() {
            database.load_userlist(userlist_path)?;
        }
    }
    Ok(())
}

fn plugin_paths(request: &SortRequest) -> Result<Vec<PathBuf>, String> {
    request
        .plugins
        .iter()
        .map(|plugin| {
            let path = PathBuf::from(plugin.path.trim());
            if path.is_file() {
                Ok(path)
            } else {
                Err(format!(
                    "plugin {:?} path does not point to a file: {}",
                    plugin.name,
                    path.display()
                ))
            }
        })
        .collect()
}

fn current_order(request: &SortRequest) -> Vec<String> {
    if request.current_order.is_empty() {
        return request
            .plugins
            .iter()
            .map(|plugin| plugin.name.trim().to_owned())
            .collect();
    }
    request
        .current_order
        .iter()
        .map(|plugin| plugin.trim().to_owned())
        .filter(|plugin| !plugin.is_empty())
        .collect()
}

fn game_type_for_id(game_id: &str) -> Option<GameType> {
    let key = game_id
        .trim()
        .to_ascii_lowercase()
        .replace(['-', '_', ' '], "");
    match key.as_str() {
        "oblivion" | "tes4" | "tesiv" => Some(GameType::Oblivion),
        "skyrim" | "tes5" | "tesv" => Some(GameType::Skyrim),
        "fallout3" | "fo3" => Some(GameType::Fallout3),
        "falloutnv" | "falloutnewvegas" | "fnv" => Some(GameType::FalloutNV),
        "fallout4" | "fo4" => Some(GameType::Fallout4),
        "skyrimse" | "skyrimspecialedition" | "sse" => Some(GameType::SkyrimSE),
        "fallout4vr" | "fo4vr" => Some(GameType::Fallout4VR),
        "skyrimvr" | "svr" => Some(GameType::SkyrimVR),
        "morrowind" | "tes3" | "tesiii" => Some(GameType::Morrowind),
        "starfield" => Some(GameType::Starfield),
        "openmw" => Some(GameType::OpenMW),
        "oblivionremastered" => Some(GameType::OblivionRemastered),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn maps_common_loot_game_ids() {
        assert_eq!(game_type_for_id("fallout4"), Some(GameType::Fallout4));
        assert_eq!(game_type_for_id("fallout4vr"), Some(GameType::Fallout4VR));
        assert_eq!(game_type_for_id("skyrimse"), Some(GameType::SkyrimSE));
        assert_eq!(
            game_type_for_id("fallout-new-vegas"),
            Some(GameType::FalloutNV)
        );
        assert_eq!(
            game_type_for_id("oblivion remastered"),
            Some(GameType::OblivionRemastered)
        );
        assert_eq!(game_type_for_id("stardewvalley"), None);
    }

    #[test]
    fn validates_contract_and_required_fields() {
        let request = SortRequest {
            game_id: "fallout4".to_owned(),
            masterlist: "masterlist.yaml".to_owned(),
            userlist: String::new(),
            prelude: String::new(),
            game_path: "/game".to_owned(),
            game_local_path: String::new(),
            plugins: vec![SortPlugin {
                name: "A.esp".to_owned(),
                path: "/game/Data/A.esp".to_owned(),
            }],
            current_order: vec!["A.esp".to_owned()],
            contract: CONTRACT.to_owned(),
        };
        assert!(validate_request(&request).is_ok());

        let bad = SortRequest {
            contract: "dmm-loot-sorter.v0".to_owned(),
            ..request
        };
        let err = validate_request(&bad).expect_err("old contract should fail");
        assert!(err.contains(CONTRACT));
    }
}
