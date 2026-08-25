import React, { useState, useRef, useEffect, useMemo } from "react";

export const SearchableSelect = ({
  options = [],
  value,
  onChange,
  placeholder = "Выберите из списка...",
  name = "",
  required = false,
  disabled = false
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [highlightedIndex, setHighlightedIndex] = useState(0);
  const containerRef = useRef(null);
  const inputRef = useRef(null);

  const selectedOption = useMemo(() => {
    if (value === undefined || value === null || value === "") return null;
    return options.find((opt) => String(opt.id) === String(value)) || null;
  }, [options, value]);

  const filteredOptions = useMemo(() => {
    const term = searchTerm.trim().toLowerCase();
    if (!term) return options;
    return options.filter((opt) => {
      const nameMatch = (opt.name || "").toLowerCase().includes(term);
      const subMatch = (opt.sub || "").toLowerCase().includes(term);
      return nameMatch || subMatch;
    });
  }, [options, searchTerm]);

  useEffect(() => {
    setHighlightedIndex(0);
  }, [filteredOptions]);

  useEffect(() => {
    const handleOutsideClick = (event) => {
      if (containerRef.current && !containerRef.current.contains(event.target)) {
        setIsOpen(false);
        setSearchTerm("");
      }
    };
    document.addEventListener("mousedown", handleOutsideClick);
    return () => document.removeEventListener("mousedown", handleOutsideClick);
  }, []);

  const handleSelect = (option) => {
    onChange(option ? option.id : "");
    setIsOpen(false);
    setSearchTerm("");
  };

  const handleClear = (e) => {
    e.stopPropagation();
    onChange("");
    setSearchTerm("");
  };

  const handleKeyDown = (e) => {
    if (disabled) return;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      if (!isOpen) {
        setIsOpen(true);
      } else {
        setHighlightedIndex((prev) => (prev + 1 < filteredOptions.length ? prev + 1 : 0));
      }
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      if (isOpen) {
        setHighlightedIndex((prev) => (prev - 1 >= 0 ? prev - 1 : filteredOptions.length - 1));
      }
    } else if (e.key === "Enter") {
      if (isOpen && filteredOptions.length > 0) {
        e.preventDefault();
        handleSelect(filteredOptions[highlightedIndex]);
      }
    } else if (e.key === "Escape") {
      setIsOpen(false);
      setSearchTerm("");
    }
  };

  return (
    <div
      className={`searchable-select ${isOpen ? "is-open" : ""} ${disabled ? "is-disabled" : ""}`}
      ref={containerRef}
      onKeyDown={handleKeyDown}
    >
      <div
        className="searchable-select-trigger"
        onClick={() => {
          if (!disabled) {
            setIsOpen(!isOpen);
            if (!isOpen && inputRef.current) {
              setTimeout(() => inputRef.current.focus(), 50);
            }
          }
        }}
      >
        {isOpen ? (
          <input
            ref={inputRef}
            type="text"
            className="searchable-select-input"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder={selectedOption ? selectedOption.name : placeholder}
            onClick={(e) => e.stopPropagation()}
            autoFocus
          />
        ) : (
          <span className={`searchable-select-label ${!selectedOption ? "is-placeholder" : ""}`}>
            {selectedOption ? selectedOption.name : placeholder}
          </span>
        )}

        <div className="searchable-select-actions">
          {selectedOption && !disabled && (
            <button
              type="button"
              className="searchable-select-clear"
              onClick={handleClear}
              title="Очистить выбор"
              aria-label="Очистить"
            >
              ✕
            </button>
          )}
          <span className="searchable-select-arrow">▾</span>
        </div>
      </div>

      <input
        type="hidden"
        name={name}
        value={value || ""}
        required={required}
      />

      {isOpen && (
        <ul className="searchable-select-dropdown" role="listbox">
          {filteredOptions.length === 0 ? (
            <li className="searchable-select-no-results">Ничего не найдено</li>
          ) : (
            filteredOptions.map((opt, index) => {
              const isSelected = selectedOption && String(selectedOption.id) === String(opt.id);
              const isHighlighted = index === highlightedIndex;

              return (
                <li
                  key={opt.id}
                  className={`searchable-select-option ${isSelected ? "is-selected" : ""} ${isHighlighted ? "is-highlighted" : ""}`}
                  onClick={() => handleSelect(opt)}
                  onMouseEnter={() => setHighlightedIndex(index)}
                  role="option"
                  aria-selected={isSelected}
                >
                  <div className="searchable-select-option-main">
                    <span className="searchable-select-option-name">{opt.name}</span>
                    {opt.sub && <span className="searchable-select-option-sub">{opt.sub}</span>}
                  </div>
                  {isSelected && <span className="searchable-select-check">✓</span>}
                </li>
              );
            })
          )}
        </ul>
      )}
    </div>
  );
};

export const SearchableMultiSelect = ({
  options = [],
  values = [],
  onChange,
  placeholder = "Поиск и добавление...",
  name = "",
  disabled = false,
  allLabel = "Выбрать все",
  clearLabel = "Очистить всё"
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [highlightedIndex, setHighlightedIndex] = useState(0);
  const containerRef = useRef(null);
  const inputRef = useRef(null);

  const selectedSet = useMemo(() => new Set((values || []).map(String)), [values]);

  const selectedOptions = useMemo(() => {
    return options.filter((opt) => selectedSet.has(String(opt.id)));
  }, [options, selectedSet]);

  const filteredOptions = useMemo(() => {
    const term = searchTerm.trim().toLowerCase();
    if (!term) return options;
    return options.filter((opt) => {
      const nameMatch = (opt.name || "").toLowerCase().includes(term);
      const subMatch = (opt.sub || "").toLowerCase().includes(term);
      return nameMatch || subMatch;
    });
  }, [options, searchTerm]);

  useEffect(() => {
    setHighlightedIndex(0);
  }, [filteredOptions]);

  useEffect(() => {
    const handleOutsideClick = (event) => {
      if (containerRef.current && !containerRef.current.contains(event.target)) {
        setIsOpen(false);
        setSearchTerm("");
      }
    };
    document.addEventListener("mousedown", handleOutsideClick);
    return () => document.removeEventListener("mousedown", handleOutsideClick);
  }, []);

  const toggleOption = (opt) => {
    const optIdStr = String(opt.id);
    let nextValues;
    if (selectedSet.has(optIdStr)) {
      nextValues = (values || []).filter((v) => String(v) !== optIdStr);
    } else {
      nextValues = [...(values || []), opt.id];
    }
    onChange(nextValues);
  };

  const removeValue = (idToRemove, e) => {
    if (e) e.stopPropagation();
    onChange((values || []).filter((v) => String(v) !== String(idToRemove)));
  };

  const handleSelectAll = (e) => {
    e.stopPropagation();
    onChange(options.map((opt) => opt.id));
  };

  const handleClearAll = (e) => {
    e.stopPropagation();
    onChange([]);
  };

  return (
    <div className={`searchable-multi-select ${isOpen ? "is-open" : ""} ${disabled ? "is-disabled" : ""}`} ref={containerRef}>
      {selectedOptions.length > 0 && (
        <div className="searchable-multi-chips">
          {selectedOptions.map((opt) => (
            <span key={opt.id} className="searchable-multi-chip">
              <span className="searchable-multi-chip-text">{opt.name}</span>
              {!disabled && (
                <button
                  type="button"
                  className="searchable-multi-chip-remove"
                  onClick={(e) => removeValue(opt.id, e)}
                  title="Удалить"
                >
                  ✕
                </button>
              )}
            </span>
          ))}
          {selectedOptions.length > 1 && !disabled && (
            <button
              type="button"
              className="searchable-multi-clear-all"
              onClick={handleClearAll}
            >
              {clearLabel}
            </button>
          )}
        </div>
      )}

      <div
        className="searchable-select-trigger"
        onClick={() => {
          if (!disabled) {
            setIsOpen(true);
            if (inputRef.current) setTimeout(() => inputRef.current.focus(), 50);
          }
        }}
      >
        <input
          ref={inputRef}
          type="text"
          className="searchable-select-input"
          value={searchTerm}
          onChange={(e) => {
            setSearchTerm(e.target.value);
            if (!isOpen) setIsOpen(true);
          }}
          onFocus={() => setIsOpen(true)}
          placeholder={selectedOptions.length === 0 ? placeholder : "Добавить еще..."}
          disabled={disabled}
        />
        <div className="searchable-select-actions">
          <span className="searchable-select-arrow">▾</span>
        </div>
      </div>

      {isOpen && (
        <div className="searchable-select-dropdown" role="listbox">
          {options.length > 0 && (
            <div className="searchable-multi-dropdown-header">
              <button type="button" className="searchable-multi-action-btn" onClick={handleSelectAll}>
                {allLabel} ({options.length})
              </button>
              {selectedOptions.length > 0 && (
                <button type="button" className="searchable-multi-action-btn is-danger" onClick={handleClearAll}>
                  {clearLabel}
                </button>
              )}
            </div>
          )}
          <ul className="searchable-multi-list">
            {filteredOptions.length === 0 ? (
              <li className="searchable-select-no-results">Ничего не найдено</li>
            ) : (
              filteredOptions.map((opt, index) => {
                const isSelected = selectedSet.has(String(opt.id));
                const isHighlighted = index === highlightedIndex;

                return (
                  <li
                    key={opt.id}
                    className={`searchable-select-option ${isSelected ? "is-selected" : ""} ${isHighlighted ? "is-highlighted" : ""}`}
                    onClick={() => toggleOption(opt)}
                    onMouseEnter={() => setHighlightedIndex(index)}
                    role="option"
                    aria-selected={isSelected}
                  >
                    <div className="searchable-select-option-main">
                      <span className="searchable-select-option-name">{opt.name}</span>
                      {opt.sub && <span className="searchable-select-option-sub">{opt.sub}</span>}
                    </div>
                    <div className="searchable-multi-checkbox">
                      {isSelected ? "✓" : ""}
                    </div>
                  </li>
                );
              })
            )}
          </ul>
        </div>
      )}
    </div>
  );
};
