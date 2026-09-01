import React from 'react';
import getAppConfig from '../../config/appConfig';
import './LegalFooter.css';

const LegalFooter = ({ onOpenLegal, className = '' }) => {
  const config = getAppConfig();
  const currentYear = new Date().getFullYear();

  return (
    <footer className={`legal-footer ${className}`}>
      <div className="legal-footer-container">
        <div className="legal-footer-links">
          <button
            type="button"
            className="legal-footer-link"
            onClick={() => onOpenLegal && onOpenLegal('privacy')}
          >
            Политика обработки ПДн (152-ФЗ)
          </button>
          <span className="legal-footer-divider">•</span>
          <button
            type="button"
            className="legal-footer-link"
            onClick={() => onOpenLegal && onOpenLegal('terms')}
          >
            Пользовательское соглашение
          </button>
          <span className="legal-footer-divider">•</span>
          <button
            type="button"
            className="legal-footer-link"
            onClick={() => onOpenLegal && onOpenLegal('copyright')}
          >
            Авторские права и DMCA
          </button>
          <span className="legal-footer-divider">•</span>
          <button
            type="button"
            className="legal-footer-link"
            onClick={() => onOpenLegal && onOpenLegal('contacts')}
          >
            Контакты
          </button>
        </div>

        <div className="legal-footer-meta">
          <p className="legal-footer-disclaimer">
            Сервис «{config.organizationName}» предоставляется в образовательных и информационных целях.
            По вопросам удаления данных или авторских прав: <a href={`mailto:${config.supportEmail}`}>{config.supportEmail}</a>
          </p>
          <p className="legal-footer-copy">
            © {currentYear} {config.domain}. Все права защищены в соответствии с законодательством РФ.
          </p>
        </div>
      </div>
    </footer>
  );
};

export default LegalFooter;
